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
	PhoneNumber string `json:"phoneNumber" jsonschema:"caller phone number"`
	Auth        string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

type startExistingPhoneVerificationInput struct {
	PhoneNumber string `json:"phoneNumber" jsonschema:"caller phone number"`
	Auth        string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

type verifyExistingPhoneOTPInput struct {
	PhoneNumber string `json:"phoneNumber" jsonschema:"caller phone number"`
	OTP         string `json:"otp" jsonschema:"verification code"`
	Auth        string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
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
		if strings.TrimSpace(in.PhoneNumber) == "" {
			return errorResult("phone number is required"), nil, nil
		}

		correlationID := auditStart(ctx, claims, "AI_TOOL_CALLED", "lookup_patient_by_phone", map[string]any{
			"phone_present": true,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.LookupPatientByPhone(ctx, &regpb.LookupPatientByPhoneRequest{
			PhoneNumber: strings.TrimSpace(in.PhoneNumber),
		})
		if err != nil {
			auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "lookup_patient_by_phone", "error", err.Error(), correlationID, nil)
			return errorResult("patient lookup failed"), nil, nil
		}

		body, _ := json.Marshal(map[string]any{
			"found":     resp.GetFound(),
			"patientID": resp.GetPatientId(),
			"fullName":  resp.GetFullName(),
		})
		auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "lookup_patient_by_phone", "success", "", correlationID, map[string]any{
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
		if strings.TrimSpace(in.PhoneNumber) == "" {
			return errorResult("phone number is required"), nil, nil
		}

		correlationID := auditStart(ctx, claims, "AI_TOOL_CALLED", "start_existing_phone_verification", nil)
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.StartExistingPhoneVerification(ctx, &regpb.StartExistingPhoneVerificationRequest{
			PhoneNumber: strings.TrimSpace(in.PhoneNumber),
		})
		if err != nil {
			auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "start_existing_phone_verification", "error", err.Error(), correlationID, nil)
			return errorResult("failed to send verification code"), nil, nil
		}
		auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "start_existing_phone_verification", "success", "", correlationID, nil)
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
		if strings.TrimSpace(in.PhoneNumber) == "" || strings.TrimSpace(in.OTP) == "" {
			return errorResult("phone number and otp are required"), nil, nil
		}

		correlationID := auditStart(ctx, claims, "AI_TOOL_CALLED", "verify_existing_phone_otp", nil)
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.VerifyExistingPhoneOTP(ctx, &regpb.VerifyExistingPhoneOTPRequest{
			PhoneNumber: strings.TrimSpace(in.PhoneNumber),
			Otp:         strings.TrimSpace(in.OTP),
		})
		if err != nil {
			auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "verify_existing_phone_otp", "error", err.Error(), correlationID, nil)
			return errorResult("failed to verify existing patient code"), nil, nil
		}

		if pid := strings.TrimSpace(resp.GetPatientId()); pid != "" {
			ensurePatientConsent(ctx, in.Auth, pid, sharedaudit.ConsentLocationAccess, "voice-otp-verification")
		}

		body, _ := json.Marshal(map[string]any{
			"message":     resp.GetMessage(),
			"patientID":   resp.GetPatientId(),
			"accessToken": resp.GetAccessToken(),
		})
		auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "verify_existing_phone_otp", "success", "", correlationID, map[string]any{
			"patient_id_present": resp.GetPatientId() != "",
		})
		return textResult(string(body)), nil, nil
	})
}
