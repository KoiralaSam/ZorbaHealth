package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	sharedlogging "github.com/KoiralaSam/ZorbaHealth/shared/logging"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

const (
	voiceOTPWaitTTL    = 5 * time.Minute
	voiceVerifiedTTL   = 10 * time.Minute
	voiceOTPFailWindow = 15 * time.Minute
	voiceOTPFailMax    = 5
)

var (
	reOTPWhole = regexp.MustCompile(`^\s*(\d{6})\s*$`)
	reOTPWord  = regexp.MustCompile(`\b(\d{6})\b`)
)

func extractVoiceOTP(body string) (string, bool) {
	body = strings.TrimSpace(body)
	if m := reOTPWhole.FindStringSubmatch(body); len(m) == 2 {
		return m[1], true
	}
	if m := reOTPWord.FindStringSubmatch(body); len(m) == 2 {
		return m[1], true
	}
	return "", false
}

func (s *PatientService) StartExistingPhoneVerification(ctx context.Context, phone string, voiceSessionID string) error {
	normalized := normalizePhone(phone)
	if normalized == "" {
		return domainErrors.ErrInvalidPhoneNumberNoDigits
	}

	patient, err := s.repo.GetPatientByPhoneNumber(ctx, normalized)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domainErrors.ErrExistingPatientNotFound
		}
		return err
	}

	otp, err := generateOTP(6)
	if err != nil {
		return domainErrors.ErrGenerateOTPFailed
	}

	otpTTL := 5 * time.Minute
	if err := s.pendingRegistrationRepo.SetOTP(ctx, normalized, "existing:"+patient.ID.String(), otp, otpTTL); err != nil {
		return domainErrors.ErrOTPSetFailed
	}

	if voiceSessionID != "" {
		if err := s.pendingRegistrationRepo.SetVoiceOTPWait(ctx, normalized, voiceSessionID, voiceOTPWaitTTL); err != nil {
			return domainErrors.ErrOTPSetFailed
		}
		s.appendVoiceAudit(ctx, sharedaudit.EventVoiceOTPWaitRegistered, normalized, voiceSessionID, map[string]any{
			"phone_hash":        sharedlogging.HashIdentifier(normalized),
			"voice_session_id":  voiceSessionID,
			"ttl_seconds":       int(voiceOTPWaitTTL.Seconds()),
		}, true, "")
	}

	if s.publisher != nil {
		if err := s.publisher.PublishPhoneVerificationCode(ctx, normalized, patient.FullName, otp); err != nil {
			return domainErrors.ErrPublishPatientCachedEventFailed
		}
	}

	return nil
}

// VerifyExistingPhoneOTPWithChannel verifies OTP and optionally records audit for voice channels.
func (s *PatientService) VerifyExistingPhoneOTPWithChannel(
	ctx context.Context,
	phone string,
	code string,
	verificationChannel string,
	voiceSessionID string,
) (*models.PatientSessionResult, error) {
	normalized := normalizePhone(phone)
	if normalized == "" {
		return nil, domainErrors.ErrInvalidPhoneNumberNoDigits
	}

	if failCount, err := s.pendingRegistrationRepo.IncrVoiceOTPFail(ctx, normalized, voiceOTPFailWindow); err == nil && failCount > voiceOTPFailMax {
		s.appendVoiceAudit(ctx, sharedaudit.EventVoiceOTPVerifyFailed, normalized, voiceSessionID, map[string]any{
			"reason":       "rate_limited",
			"phone_hash":   sharedlogging.HashIdentifier(normalized),
			"attempt_count": failCount,
		}, false, "rate_limited")
		return nil, domainErrors.ErrInvalidOrExpiredOTP
	}

	result, err := s.VerifyExistingPhoneOTP(ctx, phone, code)
	if err != nil {
		reason := "invalid_otp"
		if errors.Is(err, domainErrors.ErrInvalidOTPCode) {
			_, _ = s.pendingRegistrationRepo.IncrVoiceOTPFail(ctx, normalized, voiceOTPFailWindow)
		}
		s.appendVoiceAudit(ctx, sharedaudit.EventVoiceOTPVerifyFailed, normalized, voiceSessionID, map[string]any{
			"reason":                reason,
			"phone_hash":            sharedlogging.HashIdentifier(normalized),
			"verification_channel":  verificationChannel,
		}, false, reason)
		return nil, err
	}

	if voiceSessionID != "" && result != nil && result.PatientID != "" {
		_ = s.pendingRegistrationRepo.SetVoiceVerified(ctx, voiceSessionID, result.PatientID, voiceVerifiedTTL)
		_ = s.pendingRegistrationRepo.DeleteVoiceOTPWait(ctx, normalized)
	}

	if result != nil && result.PatientID != "" {
		meta := map[string]any{
			"verification_channel": verificationChannel,
			"voice_session_id":     voiceSessionID,
			"phone_hash":           sharedlogging.HashIdentifier(normalized),
		}
		s.appendVoiceAudit(ctx, sharedaudit.EventPatientVerified, normalized, voiceSessionID, meta, true, "")
	}

	return result, nil
}

func (s *PatientService) ProcessInboundVoiceSms(ctx context.Context, fromPhone string, messageBody string) (*models.InboundVoiceSmsResult, error) {
	correlationID := uuid.NewString()
	normalized := normalizePhone(fromPhone)
	out := &models.InboundVoiceSmsResult{CorrelationID: correlationID}

	if normalized == "" {
		out.Reason = "invalid_phone"
		s.appendVoiceAudit(ctx, sharedaudit.EventVoiceInboundSMSProcessed, normalized, "", map[string]any{
			"outcome":        "rejected",
			"reason":         out.Reason,
			"correlation_id": correlationID,
		}, false, out.Reason)
		return out, nil
	}

	voiceSessionID, err := s.pendingRegistrationRepo.GetVoiceOTPWait(ctx, normalized)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			out.Reason = "no_active_voice_session"
			s.appendVoiceAudit(ctx, sharedaudit.EventVoiceInboundSMSProcessed, normalized, "", map[string]any{
				"outcome":        "ignored",
				"reason":         out.Reason,
				"phone_hash":     sharedlogging.HashIdentifier(normalized),
				"correlation_id": correlationID,
			}, true, "")
			return out, nil
		}
		return nil, err
	}
	out.VoiceSessionID = voiceSessionID

	otp, ok := extractVoiceOTP(messageBody)
	if !ok {
		out.Reason = "no_otp_in_message"
		s.appendVoiceAudit(ctx, sharedaudit.EventVoiceInboundSMSProcessed, normalized, voiceSessionID, map[string]any{
			"outcome":        "rejected",
			"reason":         out.Reason,
			"phone_hash":     sharedlogging.HashIdentifier(normalized),
			"correlation_id": correlationID,
		}, false, out.Reason)
		return out, nil
	}

	_, err = s.VerifyExistingPhoneOTPWithChannel(ctx, normalized, otp, "sms", voiceSessionID)
	if err != nil {
		out.Reason = "verify_failed"
		s.appendVoiceAudit(ctx, sharedaudit.EventVoiceInboundSMSProcessed, normalized, voiceSessionID, map[string]any{
			"outcome":        "failed",
			"reason":         out.Reason,
			"phone_hash":     sharedlogging.HashIdentifier(normalized),
			"correlation_id": correlationID,
		}, false, out.Reason)
		return out, nil
	}

	out.Processed = true
	out.Reason = "verified"
	s.appendVoiceAudit(ctx, sharedaudit.EventVoiceInboundSMSProcessed, normalized, voiceSessionID, map[string]any{
		"outcome":        "success",
		"phone_hash":     sharedlogging.HashIdentifier(normalized),
		"correlation_id": correlationID,
	}, true, "")
	return out, nil
}

func (s *PatientService) ConsumeVoiceVerification(ctx context.Context, voiceSessionID string) (verified bool, patientID string, err error) {
	if voiceSessionID == "" {
		return false, "", nil
	}
	pid, err := s.pendingRegistrationRepo.ConsumeVoiceVerified(ctx, voiceSessionID)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, pid, nil
}

// BindCallerPhone returns normalized phone for MCP binding checks.
func BindCallerPhone(phone string) string {
	return sharedauth.NormalizePhoneDigits(phone)
}
