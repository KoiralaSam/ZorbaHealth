package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedenv "github.com/KoiralaSam/ZorbaHealth/shared/env"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// E.164-ish: optional +, then 10–15 digits.
var phoneRegex = regexp.MustCompile(`^\+?[0-9]{10,15}$`)

type PatientService struct {
	repo                    outbound.PatientRepository
	welfareChecks           outbound.WelfareCheckRepository
	authService             outbound.AuthRepository
	pendingRegistrationRepo outbound.PendingRegistrationRepository
	publisher               outbound.PatientPublisher
	welfareCalls            outbound.WelfareCheckCallProvider
	audit                   auditpb.AuditServiceClient
}

func NewPatientService(
	repo outbound.PatientRepository,
	authService outbound.AuthRepository,
	pendingRegistrationRepo outbound.PendingRegistrationRepository,
	publisher outbound.PatientPublisher,
	audit auditpb.AuditServiceClient,
	welfareCalls outbound.WelfareCheckCallProvider,
) *PatientService {
	welfareRepo, _ := repo.(outbound.WelfareCheckRepository)
	return &PatientService{
		repo:                    repo,
		welfareChecks:           welfareRepo,
		authService:             authService,
		pendingRegistrationRepo: pendingRegistrationRepo,
		publisher:               publisher,
		welfareCalls:            welfareCalls,
		audit:                   audit,
	}
}

func (s *PatientService) StartRegistrationWithVerification(ctx context.Context, req *models.RegisterPatientRequest) (verificationToken string, otp string, err error) {
	if req == nil {
		return "", "", domainErrors.ErrRegistrationRequestRequired
	}
	if !phoneRegex.MatchString(req.PhoneNumber) {
		return "", "", domainErrors.ErrInvalidPhoneNumber
	}
	req.PhoneNumber = normalizePhone(req.PhoneNumber)
	if req.DateOfBirth.IsZero() {
		return "", "", domainErrors.ErrDateOfBirthRequired
	}
	if req.DateOfBirth.After(time.Now()) {
		return "", "", domainErrors.ErrDateOfBirthInFuture
	}

	otp, err = generateOTP(6)
	if err != nil {
		return "", "", domainErrors.ErrGenerateOTPFailed
	}

	token := uuid.New().String()
	pendingRegistration := &models.PendingRegistration{
		Email:         req.Email,
		PhoneNumber:   req.PhoneNumber,
		Password:      req.Password,
		FullName:      req.FullName,
		DateOfBirth:   req.DateOfBirth,
		CreatedAt:     time.Now(),
		PhoneVerified: false,
		EmailVerified: false,
	}
	ttl := 15 * time.Minute
	if err := s.pendingRegistrationRepo.Set(ctx, token, pendingRegistration, ttl); err != nil {
		return "", "", domainErrors.ErrPendingRegistrationSetFailed
	}
	otpTTL := 5 * time.Minute
	if err := s.pendingRegistrationRepo.SetOTP(ctx, req.PhoneNumber, token, otp, otpTTL); err != nil {
		return "", "", domainErrors.ErrOTPSetFailed
	}
	if s.publisher != nil {
		if err := s.publisher.PublishPatientChached(ctx, req, token, otp); err != nil {
			return "", "", domainErrors.ErrPublishPatientCachedEventFailed
		}
	}
	return token, otp, nil
}

func generateOTP(digits int) (string, error) {
	const digitset = "0123456789"
	b := make([]byte, digits)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = digitset[int(b[i])%len(digitset)]
	}
	return string(b), nil
}

func (s *PatientService) VerifyEmailAndCreatePatient(ctx context.Context, token string) (*models.Patient, error) {
	pending, err := s.pendingRegistrationRepo.Get(ctx, token)
	if err != nil {
		return nil, domainErrors.ErrInvalidOrExpiredVerificationLink
	}

	// Mark email as verified (user clicked the email link)
	pending.EmailVerified = true
	ttl := 15 * time.Minute
	if err := s.pendingRegistrationRepo.Set(ctx, token, pending, ttl); err != nil {
		return nil, domainErrors.ErrPendingRegistrationUpdateFailed
	}

	if !pending.PhoneVerified {
		return nil, domainErrors.ErrPhoneVerificationRequired
	}

	// Both verified: create patient and clean up
	defer s.pendingRegistrationRepo.Delete(ctx, token)
	req := &models.RegisterPatientRequest{
		PhoneNumber: pending.PhoneNumber,
		Email:       pending.Email,
		Password:    pending.Password,
		FullName:    pending.FullName,
		DateOfBirth: pending.DateOfBirth,
	}

	authResult, err := s.authService.RegisterPatient(ctx, req)
	if err != nil {
		return nil, domainErrors.ErrAuthServiceRegisterPatientFailed
	}

	userID, err := uuid.Parse(authResult.UserID)
	if err != nil {
		return nil, domainErrors.ErrAuthServiceInvalidUserID
	}

	patient := &models.Patient{
		UserID:      userID,
		PhoneNumber: pending.PhoneNumber,
		Email:       pending.Email,
		FullName:    pending.FullName,
		DateOfBirth: pending.DateOfBirth,
	}
	patient, err = s.repo.CreatePatient(ctx, patient)
	if err != nil {
		return nil, domainErrors.ErrPatientCreationFailed
	}
	if s.publisher != nil {
		if err := s.publisher.PublishPatientRegistered(ctx, patient); err != nil {
			return nil, domainErrors.ErrPublishPatientRegisteredEventFailed
		}
	}
	return patient, nil
}

// VerifyPhoneOTP verifies the OTP for the given phone and sets PhoneVerified on the pending registration.
func (s *PatientService) VerifyPhoneOTP(ctx context.Context, phone string, code string) error {
	normalized := normalizePhone(phone)
	if normalized == "" {
		return domainErrors.ErrInvalidPhoneNumberNoDigits
	}

	storedToken, storedCode, err := s.pendingRegistrationRepo.GetOTP(ctx, normalized)
	if err != nil {
		return domainErrors.ErrInvalidOrExpiredOTP
	}
	if storedCode != code {
		return domainErrors.ErrInvalidOTPCode
	}
	if strings.HasPrefix(storedToken, "existing:") {
		return domainErrors.ErrExistingPatientVerificationState
	}

	pending, err := s.pendingRegistrationRepo.Get(ctx, storedToken)
	if err != nil {
		return domainErrors.ErrPendingRegistrationNotFoundOrExpired
	}
	pending.PhoneVerified = true
	ttl := 15 * time.Minute
	if err := s.pendingRegistrationRepo.Set(ctx, storedToken, pending, ttl); err != nil {
		return domainErrors.ErrPendingRegistrationUpdateFailed
	}
	_ = s.pendingRegistrationRepo.DeleteOTP(ctx, normalized)
	return nil
}

func (s *PatientService) VerifyExistingPhoneOTP(ctx context.Context, phone string, code string) (*models.PatientSessionResult, error) {
	normalized := normalizePhone(phone)
	if normalized == "" {
		return nil, domainErrors.ErrInvalidPhoneNumberNoDigits
	}

	storedToken, storedCode, err := s.pendingRegistrationRepo.GetOTP(ctx, normalized)
	if err != nil {
		return nil, domainErrors.ErrInvalidOrExpiredOTP
	}
	if storedCode != code {
		return nil, domainErrors.ErrInvalidOTPCode
	}
	if !strings.HasPrefix(storedToken, "existing:") {
		return nil, domainErrors.ErrExistingPatientVerificationState
	}

	patientID := strings.TrimPrefix(storedToken, "existing:")
	if patientID == "" {
		return nil, domainErrors.ErrExistingPatientVerificationState
	}

	_ = s.pendingRegistrationRepo.DeleteOTP(ctx, normalized)
	patient, err := s.repo.GetPatientByID(ctx, patientID)
	if err != nil {
		return nil, err
	}
	session, err := s.authService.CreatePatientSession(ctx, patient.UserID.String(), []string{"location:read", "records:read"})
	if err != nil {
		return nil, domainErrors.ErrAuthServiceRegisterPatientFailed
	}
	_ = session
	return &models.PatientSessionResult{
		PatientID:   patient.ID.String(),
		UserID:      patient.UserID.String(),
		AccessToken: session.AccessToken,
	}, nil
}

func (s *PatientService) LoginPatient(
	ctx context.Context,
	patient *models.Patient,
) (*models.PatientSessionResult, error) {
	if patient == nil {
		return nil, domainErrors.ErrRegistrationRequestRequired
	}

	userID, _, err := s.authService.ValidateUserCredentials(ctx, &models.LoginRequest{
		Email:       strings.TrimSpace(patient.Email),
		PhoneNumber: normalizePhone(patient.PhoneNumber),
		Password:    patient.MedicalNotes,
	})
	if err != nil {
		return nil, err
	}

	patientRecord, err := s.repo.GetPatientByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	session, err := s.authService.CreatePatientSession(ctx, patientRecord.UserID.String(), []string{"location:read", "records:read"})
	if err != nil {
		return nil, err
	}

	return &models.PatientSessionResult{
		PatientID:    patientRecord.ID.String(),
		UserID:       patientRecord.UserID.String(),
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
	}, nil
}

func (s *PatientService) GetPatientByID(
	ctx context.Context,
	id string,
) (*models.Patient, error) {
	// business rules can live here
	return s.repo.GetPatientByID(ctx, id)
}

func (s *PatientService) GetPatientProfile(ctx context.Context, patientID string) (*models.PatientProfile, error) {
	return s.repo.GetPatientProfile(ctx, patientID)
}

func (s *PatientService) ListPatientCallSummaries(ctx context.Context, patientID string, limit, offset int32) ([]models.CallSummary, error) {
	return s.repo.ListPatientCallSummaries(ctx, patientID, limit, offset)
}

func (s *PatientService) CreateWelfareCheck(ctx context.Context, cmd *models.CreateWelfareCheckCommand) (*models.WelfareCheck, error) {
	if cmd == nil {
		return nil, domainErrors.ErrRegistrationRequestRequired
	}
	if s.welfareChecks == nil {
		return nil, domainErrors.ErrWelfareCheckDispatchUnavailable
	}
	if cmd.PatientID == uuid.Nil {
		return nil, domainErrors.ErrWelfareCheckNotFound
	}
	now := time.Now().UTC()
	if !cmd.ScheduledAt.After(now.Add(1 * time.Minute)) {
		return nil, domainErrors.ErrWelfareCheckStartsAtInvalid
	}
	if cmd.ScheduledAt.After(now.Add(90 * 24 * time.Hour)) {
		return nil, domainErrors.ErrWelfareCheckStartsAtInvalid
	}
	timezone := strings.TrimSpace(cmd.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, domainErrors.ErrWelfareCheckTimezoneInvalid
	}
	reason := models.WelfareCheckReason(strings.TrimSpace(string(cmd.ReasonCode)))
	if !isValidWelfareReason(reason) {
		return nil, domainErrors.ErrWelfareCheckReasonInvalid
	}
	detail := strings.TrimSpace(cmd.ReasonDetail)
	if len(detail) > 1000 {
		return nil, domainErrors.ErrWelfareCheckReasonTooLong
	}
	patient, err := s.repo.GetPatientByID(ctx, cmd.PatientID.String())
	if err != nil {
		return nil, err
	}
	if normalizePhone(patient.PhoneNumber) == "" {
		return nil, domainErrors.ErrWelfareCheckPhoneRequired
	}
	if err := s.requireWelfareCheckConsents(ctx, cmd.PatientID.String()); err != nil {
		return nil, err
	}
	check := &models.WelfareCheck{
		PatientID:    cmd.PatientID,
		ScheduledAt:  cmd.ScheduledAt.UTC(),
		Timezone:     timezone,
		ReasonCode:   reason,
		ReasonDetail: detail,
		Status:       models.WelfareCheckStatusScheduled,
	}
	saved, err := s.welfareChecks.InsertWelfareCheck(ctx, check)
	if err != nil {
		return nil, err
	}
	s.appendWelfareAudit(ctx, sharedaudit.EventWelfareCheckRequested, saved.PatientID.String(), map[string]any{
		"welfare_check_id": saved.ID.String(),
		"reason_code":      string(saved.ReasonCode),
		"scheduled_at":     saved.ScheduledAt.Format(time.RFC3339),
	}, true, "")
	return saved, nil
}

func (s *PatientService) ListWelfareChecks(ctx context.Context, filter models.ListWelfareChecksFilter) ([]models.WelfareCheck, error) {
	if s.welfareChecks == nil {
		return nil, domainErrors.ErrWelfareCheckDispatchUnavailable
	}
	return s.welfareChecks.ListWelfareChecks(ctx, filter)
}

func (s *PatientService) CancelWelfareCheck(ctx context.Context, patientID string, welfareCheckID string) (*models.WelfareCheck, error) {
	if s.welfareChecks == nil {
		return nil, domainErrors.ErrWelfareCheckDispatchUnavailable
	}
	pid, err := uuid.Parse(strings.TrimSpace(patientID))
	if err != nil {
		return nil, domainErrors.ErrWelfareCheckNotFound
	}
	cid, err := uuid.Parse(strings.TrimSpace(welfareCheckID))
	if err != nil {
		return nil, domainErrors.ErrWelfareCheckNotFound
	}
	check, err := s.welfareChecks.CancelWelfareCheck(ctx, pid, cid)
	if err != nil {
		return nil, err
	}
	s.appendWelfareAudit(ctx, sharedaudit.EventWelfareCheckCancelled, patientID, map[string]any{
		"welfare_check_id": welfareCheckID,
	}, true, "")
	return check, nil
}

func (s *PatientService) DispatchDueWelfareChecks(ctx context.Context, limit int32) ([]models.WelfareCheckDispatchResult, error) {
	if s.welfareChecks == nil || s.welfareCalls == nil {
		return nil, domainErrors.ErrWelfareCheckDispatchUnavailable
	}
	runs, err := s.welfareChecks.ClaimDueWelfareCheckRuns(ctx, limit)
	if err != nil {
		return nil, err
	}
	results := make([]models.WelfareCheckDispatchResult, 0, len(runs))
	var firstErr error
	failed := 0
	for _, run := range runs {
		result, err := s.dispatchWelfareCheckRun(ctx, run)
		if err != nil {
			failed++
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		results = append(results, *result)
	}
	if len(runs) > 0 && len(results) == 0 && firstErr != nil {
		// All claimed runs failed at the provider/service layer; surface for Job failure.
		return results, fmt.Errorf("all %d welfare check run(s) failed: %w", len(runs), firstErr)
	}
	_ = failed
	return results, nil
}

func (s *PatientService) dispatchWelfareCheckRun(ctx context.Context, run models.WelfareCheckRun) (*models.WelfareCheckDispatchResult, error) {
	if err := s.requireWelfareCheckConsents(ctx, run.PatientID.String()); err != nil {
		_ = s.welfareChecks.MarkWelfareRunFailed(ctx, run.ID, err.Error(), false)
		s.appendWelfareAudit(ctx, sharedaudit.EventWelfareCheckFailed, run.PatientID.String(), welfareRunAuditMeta(run), false, err.Error())
		return nil, err
	}
	if normalizePhone(run.PatientPhoneNumber) == "" {
		_ = s.welfareChecks.MarkWelfareRunFailed(ctx, run.ID, domainErrors.ErrWelfareCheckPhoneRequired.Error(), false)
		return nil, domainErrors.ErrWelfareCheckPhoneRequired
	}

	// Idempotent recovery: LiveKit may have succeeded while DB mark failed.
	if strings.TrimSpace(run.LiveKitDispatchID) != "" || strings.TrimSpace(run.LiveKitSIPCallID) != "" {
		roomName := strings.TrimSpace(run.LiveKitRoomName)
		if roomName == "" {
			roomName = "welfare-check-" + run.ID.String()
		}
		result := models.WelfareCheckDispatchResult{
			RunID:     run.ID,
			RequestID: run.RequestID,
			RoomName:  roomName,
			RoomSID:   run.LiveKitRoomSID,
			DispatchID: run.LiveKitDispatchID,
			SIPCallID:  run.LiveKitSIPCallID,
		}
		if err := s.welfareChecks.MarkWelfareRunDispatched(ctx, result); err != nil {
			return nil, err
		}
		return &result, nil
	}

	session, err := s.authService.CreatePatientSession(ctx, run.PatientUserID.String(), []string{"location:read", "records:read"})
	if err != nil {
		_ = s.welfareChecks.MarkWelfareRunFailed(ctx, run.ID, err.Error(), run.Attempts < 3)
		return nil, err
	}
	roomName := strings.TrimSpace(run.LiveKitRoomName)
	if roomName == "" {
		roomName = "welfare-check-" + run.ID.String()
	}
	call, err := s.welfareCalls.StartWelfareCheckCall(ctx, outbound.WelfareCheckCallInput{
		RequestID:    run.RequestID.String(),
		RunID:        run.ID.String(),
		RoomName:     roomName,
		PatientID:    run.PatientID.String(),
		PatientName:  run.PatientFullName,
		PatientPhone: run.PatientPhoneNumber,
		ScheduledAt:  run.ScheduledAt,
		Timezone:     run.RequestTimezone,
		ReasonCode:   string(run.RequestReasonCode),
		ReasonDetail: run.RequestReasonDetail,
		PatientToken: session.AccessToken,
		AgentName:    "zorba-health-voice",
	})
	if err != nil {
		if run.Attempts >= 3 {
			_ = s.welfareChecks.MarkWelfareRunMissed(ctx, run.ID, err.Error())
			s.appendWelfareAudit(ctx, sharedaudit.EventWelfareCheckMissed, run.PatientID.String(), welfareRunAuditMeta(run), false, err.Error())
		} else {
			_ = s.welfareChecks.MarkWelfareRunFailed(ctx, run.ID, err.Error(), true)
			s.appendWelfareAudit(ctx, sharedaudit.EventWelfareCheckFailed, run.PatientID.String(), welfareRunAuditMeta(run), false, err.Error())
		}
		return nil, err
	}
	result := models.WelfareCheckDispatchResult{
		RunID:               run.ID,
		RequestID:           run.RequestID,
		RoomName:            call.RoomName,
		RoomSID:             call.RoomSID,
		DispatchID:          call.DispatchID,
		SIPCallID:           call.SIPCallID,
		ParticipantID:       call.ParticipantID,
		ParticipantIdentity: call.ParticipantIdentity,
	}
	// Persist LiveKit identity before status transition so reclaim can skip duplicate outbound calls.
	_ = s.welfareChecks.PersistWelfareRunLiveKitResult(ctx, result)
	if err := s.welfareChecks.MarkWelfareRunDispatched(ctx, result); err != nil {
		return nil, err
	}
	meta := welfareRunAuditMeta(run)
	meta["livekit_room_name"] = result.RoomName
	meta["livekit_sip_call_id"] = result.SIPCallID
	s.appendWelfareAudit(ctx, sharedaudit.EventWelfareCheckDispatched, run.PatientID.String(), meta, true, "")
	s.appendWelfareAudit(ctx, sharedaudit.EventWelfareCheckRecordAccessed, run.PatientID.String(), meta, true, "")
	return &result, nil
}

func (s *PatientService) UpdateWelfareRunLifecycle(ctx context.Context, cmd *models.UpdateWelfareRunLifecycleCommand) (*models.WelfareCheckRun, error) {
	if cmd == nil {
		return nil, domainErrors.ErrRegistrationRequestRequired
	}
	if s.welfareChecks == nil {
		return nil, domainErrors.ErrWelfareCheckDispatchUnavailable
	}
	patientID, err := uuid.Parse(strings.TrimSpace(cmd.PatientID))
	if err != nil {
		return nil, domainErrors.ErrWelfareCheckRunNotFound
	}
	runID, err := uuid.Parse(strings.TrimSpace(cmd.RunID))
	if err != nil {
		return nil, domainErrors.ErrWelfareCheckRunNotFound
	}
	status := models.WelfareCheckRunStatus(strings.TrimSpace(string(cmd.Status)))
	switch status {
	case models.WelfareRunStatusAnswered,
		models.WelfareRunStatusCompleted,
		models.WelfareRunStatusMissed,
		models.WelfareRunStatusFailed:
	default:
		return nil, domainErrors.ErrWelfareCheckRunTransition
	}
	run, err := s.welfareChecks.UpdateWelfareRunLifecycle(ctx, patientID, runID, status, strings.TrimSpace(cmd.Reason))
	if err != nil {
		return nil, err
	}
	eventType := sharedaudit.EventWelfareCheckFailed
	success := false
	switch status {
	case models.WelfareRunStatusAnswered:
		eventType = sharedaudit.EventWelfareCheckAnswered
		success = true
	case models.WelfareRunStatusCompleted:
		eventType = sharedaudit.EventWelfareCheckCompleted
		success = true
	case models.WelfareRunStatusMissed:
		eventType = sharedaudit.EventWelfareCheckMissed
	case models.WelfareRunStatusFailed:
		eventType = sharedaudit.EventWelfareCheckFailed
	}
	s.appendWelfareAudit(ctx, eventType, patientID.String(), welfareRunAuditMeta(*run), success, strings.TrimSpace(cmd.Reason))
	return run, nil
}

func (s *PatientService) requireWelfareCheckConsents(ctx context.Context, patientID string) error {
	if s.audit == nil {
		if sharedenv.GetBool("WELFARE_CHECK_ALLOW_CONSENT_BYPASS", false) {
			return nil
		}
		return domainErrors.ErrWelfareCheckConsentUnavailable
	}
	for _, consentType := range []string{sharedaudit.ConsentVoiceAssistantUse, sharedaudit.ConsentHealthRecordAccess} {
		resp, err := s.audit.CheckConsent(ctx, &auditpb.CheckConsentRequest{
			PatientId:   patientID,
			ConsentType: consentType,
			Scope:       "",
		})
		if err != nil {
			if sharedenv.GetBool("WELFARE_CHECK_ALLOW_CONSENT_BYPASS", false) {
				return nil
			}
			return fmt.Errorf("%w: %v", domainErrors.ErrWelfareCheckConsentUnavailable, err)
		}
		if !resp.GetAllowed() {
			if reason := strings.TrimSpace(resp.GetDenialReason()); reason != "" {
				return fmt.Errorf("%w: %s", domainErrors.ErrWelfareCheckConsentRequired, reason)
			}
			return fmt.Errorf("%w: %s", domainErrors.ErrWelfareCheckConsentRequired, consentType)
		}
	}
	return nil
}

func isValidWelfareReason(reason models.WelfareCheckReason) bool {
	switch reason {
	case models.WelfareReasonMedicationReminder,
		models.WelfareReasonMentalWellbeing,
		models.WelfareReasonDailyCheckup,
		models.WelfareReasonSymptomFollowUp,
		models.WelfareReasonCarePlanReminder,
		models.WelfareReasonOther:
		return true
	default:
		return false
	}
}

func welfareRunAuditMeta(run models.WelfareCheckRun) map[string]any {
	return map[string]any{
		"welfare_check_id": run.RequestID.String(),
		"welfare_run_id":   run.ID.String(),
		"reason_code":      string(run.RequestReasonCode),
		"scheduled_at":     run.ScheduledAt.Format(time.RFC3339),
		"attempts":         run.Attempts,
	}
}

func (s *PatientService) appendWelfareAudit(ctx context.Context, eventType string, patientID string, metadata map[string]any, success bool, failure string) {
	if s.audit == nil {
		return
	}
	meta, err := structpb.NewStruct(metadata)
	if err != nil {
		meta = &structpb.Struct{}
	}
	_, _ = s.audit.AppendAuditEvent(ctx, &auditpb.AppendAuditEventRequest{
		Event: &auditpb.AuditEvent{
			EventId:       uuid.NewString(),
			EventType:     eventType,
			ActorType:     "patient",
			ActorId:       patientID,
			PatientId:     patientID,
			ServiceName:   "patient-service",
			Timestamp:     timestamppb.Now(),
			SuccessStatus: success,
			FailureReason: failure,
			Metadata:      meta,
		},
	})
}

// normalizePhone returns digits only (E.164 without +) for consistent lookup.
func normalizePhone(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *PatientService) GetPatientByPhoneNumber(
	ctx context.Context,
	phoneNumber string,
) (*models.Patient, error) {
	normalized := normalizePhone(phoneNumber)
	if normalized == "" {
		return nil, domainErrors.ErrInvalidPhoneNumberNoDigits
	}
	return s.repo.GetPatientByPhoneNumber(ctx, normalized)
}

func (s *PatientService) GetPatientByEmail(
	ctx context.Context,
	email string,
) (*models.Patient, error) {
	normalized := strings.TrimSpace(strings.ToLower(email))
	if normalized == "" {
		return nil, domainErrors.ErrEmailRequired
	}
	return s.repo.GetPatientByEmail(ctx, normalized)
}

func (s *PatientService) UpdatePatient(
	ctx context.Context,
	patient *models.Patient,
) error {
	// business rules can live here
	// e.g., validate updated data, check permissions, audit logging
	return s.repo.UpdatePatient(ctx, patient)
}

func (s *PatientService) DeletePatient(
	ctx context.Context,
	id string,
) error {
	// business rules can live here
	// e.g., check if patient has active calls, cascade delete related data
	return s.repo.DeletePatient(ctx, id)
}
