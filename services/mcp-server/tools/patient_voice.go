package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	regpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/registration_verification"
)

type lookupPatientByPhoneInput struct {
	PhoneNumber               string `json:"phoneNumber" jsonschema:"caller phone number"`
	Auth                      string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
	VerificationCorrelationID string `json:"_verificationCorrelationId,omitempty"`
}

type startExistingPhoneVerificationInput struct {
	PhoneNumber               string `json:"phoneNumber" jsonschema:"caller phone number"`
	Auth                      string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
	VerificationCorrelationID string `json:"_verificationCorrelationId,omitempty"`
}

type verifyExistingPhoneOTPInput struct {
	PhoneNumber               string `json:"phoneNumber" jsonschema:"caller phone number"`
	OTP                       string `json:"otp" jsonschema:"verification code"`
	Auth                      string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
	VerificationChannel       string `json:"verificationChannel,omitempty"`
	VerificationCorrelationID string `json:"_verificationCorrelationId,omitempty"`
}

type consumeVoiceVerificationInput struct {
	Auth                      string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
	VerificationChannel       string `json:"verificationChannel,omitempty"`
	VerificationCorrelationID string `json:"_verificationCorrelationId,omitempty"`
}

func RegisterLookupPatientByPhone(s *mcp.Server, db *pgxpool.Pool, client regpb.RegistrationVerificationServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "lookup_patient_by_phone",
		Description: "Check whether the caller phone already belongs to a patient",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in lookupPatientByPhoneInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		phone, err := boundVoicePhone(claims, in.PhoneNumber)
		if err != nil {
			auditCompat(db, claims, "lookup_patient_by_phone", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}

		meta := voiceAuditMeta(claims, "", in.VerificationCorrelationID)
		meta["phone_present"] = true
		correlationID := auditStart(ctx, claims, sharedaudit.EventAIToolCalled, "lookup_patient_by_phone", meta)
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.LookupPatientByPhone(ctx, &regpb.LookupPatientByPhoneRequest{
			PhoneNumber: phone,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "lookup_patient_by_phone", "error", err.Error(), correlationID, nil)
			return errorResult("patient lookup failed"), nil, nil
		}

		body, _ := json.Marshal(map[string]any{
			"found":     resp.GetFound(),
			"patientID": resp.GetPatientId(),
			"fullName":  resp.GetFullName(),
		})
		auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "lookup_patient_by_phone", "success", "", correlationID, map[string]any{
			"found": resp.GetFound(),
		})
		return textResult(string(body)), nil, nil
	})
}

func RegisterStartExistingPhoneVerification(s *mcp.Server, db *pgxpool.Pool, client regpb.RegistrationVerificationServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "start_existing_phone_verification",
		Description: "Send an OTP to an existing patient on the caller phone number",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in startExistingPhoneVerificationInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		phone, err := boundVoicePhone(claims, in.PhoneNumber)
		if err != nil {
			auditCompat(db, claims, "start_existing_phone_verification", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}

		meta := voiceAuditMeta(claims, "", in.VerificationCorrelationID)
		correlationID := auditStart(ctx, claims, sharedaudit.EventAIToolCalled, "start_existing_phone_verification", meta)
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.StartExistingPhoneVerification(ctx, &regpb.StartExistingPhoneVerificationRequest{
			PhoneNumber:    phone,
			VoiceSessionId: claims.SessionID,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "start_existing_phone_verification", "error", err.Error(), correlationID, nil)
			return errorResult("failed to send verification code"), nil, nil
		}
		auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "start_existing_phone_verification", "success", "", correlationID, nil)
		return textResult(resp.GetMessage()), nil, nil
	})
}

func RegisterVerifyExistingPhoneOTP(s *mcp.Server, db *pgxpool.Pool, client regpb.RegistrationVerificationServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "verify_existing_phone_otp",
		Description: "Verify an existing patient by SMS OTP and return the patient ID",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in verifyExistingPhoneOTPInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		phone, err := boundVoicePhone(claims, in.PhoneNumber)
		if err != nil {
			auditCompat(db, claims, "verify_existing_phone_otp", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		otp := strings.TrimSpace(in.OTP)
		if !validateOTPFormat(otp) {
			auditCompat(db, claims, "verify_existing_phone_otp", "error", "invalid otp format")
			return errorResult("invalid verification code format"), nil, nil
		}

		channel := strings.TrimSpace(in.VerificationChannel)
		if channel == "" {
			channel = "mcp_tool"
		}
		meta := voiceAuditMeta(claims, channel, in.VerificationCorrelationID)
		correlationID := auditStart(ctx, claims, sharedaudit.EventAIToolCalled, "verify_existing_phone_otp", meta)
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.VerifyExistingPhoneOTP(ctx, &regpb.VerifyExistingPhoneOTPRequest{
			PhoneNumber: phone,
			Otp:         otp,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventVoiceOTPVerifyFailed, "verify_existing_phone_otp", "error", err.Error(), correlationID, meta)
			return errorResult("failed to verify existing patient code"), nil, nil
		}

		if pid := strings.TrimSpace(resp.GetPatientId()); pid != "" {
			ensurePatientConsent(ctx, in.Auth, pid, sharedaudit.ConsentLocationAccess, "voice-otp-verification")
			auditComplete(ctx, db, claims, sharedaudit.EventPatientVerified, "verify_existing_phone_otp", "success", "", correlationID, meta)
		} else {
			auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "verify_existing_phone_otp", "success", "", correlationID, meta)
		}

		body, _ := json.Marshal(map[string]any{
			"message":     resp.GetMessage(),
			"patientID":   resp.GetPatientId(),
			"accessToken": resp.GetAccessToken(),
		})
		return textResult(string(body)), nil, nil
	})
}

func RegisterConsumeVoiceVerification(s *mcp.Server, db *pgxpool.Pool, client regpb.RegistrationVerificationServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "consume_voice_verification",
		Description: "Poll for patient verification completed via inbound SMS during this voice session",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in consumeVoiceVerificationInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		if claims.SessionID == "" {
			return errorResult("session id is required"), nil, nil
		}

		channel := strings.TrimSpace(in.VerificationChannel)
		if channel == "" {
			channel = "sms_poll"
		}
		meta := voiceAuditMeta(claims, channel, in.VerificationCorrelationID)
		correlationID := auditStart(ctx, claims, sharedaudit.EventAIToolCalled, "consume_voice_verification", meta)
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.ConsumeVoiceVerification(ctx, &regpb.ConsumeVoiceVerificationRequest{
			VoiceSessionId: claims.SessionID,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "consume_voice_verification", "error", err.Error(), correlationID, nil)
			return errorResult("failed to consume voice verification"), nil, nil
		}

		body, _ := json.Marshal(map[string]any{
			"verified":  resp.GetVerified(),
			"patientID": resp.GetPatientId(),
		})
		outcome := "pending"
		if resp.GetVerified() {
			outcome = "success"
		}
		auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "consume_voice_verification", outcome, "", correlationID, map[string]any{
			"verified": resp.GetVerified(),
		})
		return textResult(string(body)), nil, nil
	})
}
