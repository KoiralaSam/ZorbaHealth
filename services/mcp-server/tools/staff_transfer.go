package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	schedpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/scheduling"
)

type requestStaffTransferInput struct {
	SessionID      string `json:"sessionID" jsonschema:"voice session / LiveKit room SID"`
	RoomSID        string `json:"roomSID,omitempty" jsonschema:"optional LiveKit room SID (defaults to sessionID)"`
	HospitalID     string `json:"hospitalID,omitempty" jsonschema:"optional hospital UUID; auto-picked when the patient has exactly one consented hospital"`
	StaffID        string `json:"staffID,omitempty" jsonschema:"optional preferred staff UUID"`
	TransferReason string `json:"transferReason,omitempty" jsonschema:"short reason for connecting hospital staff"`
	PatientLang    string `json:"patientLanguage,omitempty" jsonschema:"ISO 639-1 language the patient wants to speak during interpretation"`
	Auth           string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterRequestStaffTransfer(s *mcp.Server, db *pgxpool.Pool, client schedpb.SchedulingServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "request_staff_transfer",
		Description: "Request that hospital staff join the patient's active LiveKit SIP call for live interpretation. " +
			"Use after identity verification when the caller presses 0, asks to speak with hospital staff, or needs a clinician on the line. " +
			"hospitalID may be omitted when the patient has exactly one consented hospital.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in requestStaffTransferInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		if err := sharedauth.RequireActorType(claims, sharedauth.ActorPatient); err != nil {
			auditCompat(db, claims, "request_staff_transfer", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		if claims.PatientID == "" || strings.HasPrefix(claims.PatientID, "session:") {
			auditCompat(db, claims, "request_staff_transfer", "forbidden", "verified patient required")
			return errorResult("identity verification is required before connecting hospital staff"), nil, nil
		}
		sessionID := strings.TrimSpace(in.SessionID)
		if sessionID == "" {
			sessionID = strings.TrimSpace(claims.SessionID)
		}
		if sessionID == "" {
			return errorResult("sessionID is required"), nil, nil
		}

		hospitalID := strings.TrimSpace(in.HospitalID)
		hospitals, err := queryActivePatientHospitals(ctx, db, claims.PatientID)
		if err != nil {
			return errorResult("failed to look up hospital consent"), nil, nil
		}
		if hospitalID == "" {
			switch len(hospitals) {
			case 0:
				return errorResult("No hospital consent found. Grant hospital data-sharing consent in the Zorba patient portal, then try again."), nil, nil
			case 1:
				hospitalID = hospitals[0].HospitalID
			default:
				names := make([]string, 0, len(hospitals))
				for _, h := range hospitals {
					names = append(names, h.HospitalName+" ("+h.HospitalID+")")
				}
				body, _ := json.Marshal(map[string]any{
					"error":     "hospitalID is required when the patient has multiple consented hospitals",
					"hospitals": hospitals,
					"message":   "Ask the caller which hospital to connect, then call again with hospitalID.",
					"options":   names,
				})
				return textResult(string(body)), nil, nil
			}
		} else if msg := hospitalConsentMismatchMessage(ctx, db, claims.PatientID, hospitalID, hospitals); msg != "" {
			return errorResult(msg), nil, nil
		}

		roomSID := strings.TrimSpace(in.RoomSID)
		if roomSID == "" {
			roomSID = sessionID
		}
		reason := strings.TrimSpace(in.TransferReason)
		if reason == "" {
			reason = "patient_requested_staff"
		}

		correlationID := auditStart(ctx, claims, sharedaudit.EventAIToolCalled, "request_staff_transfer", map[string]any{
			"session_id":  sessionID,
			"hospital_id": hospitalID,
			"staff_id":    strings.TrimSpace(in.StaffID),
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.RequestBridgedCallTransfer(ctx, &schedpb.RequestBridgedCallTransferRequest{
			SessionId:      sessionID,
			RoomSid:        roomSID,
			PatientId:      claims.PatientID,
			HospitalId:     hospitalID,
			StaffId:        strings.TrimSpace(in.StaffID),
			TransferReason: reason,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "request_staff_transfer", "error", err.Error(), correlationID, nil)
			return errorResult("failed to request staff transfer: " + err.Error()), nil, nil
		}

		patientLang := strings.TrimSpace(strings.ToLower(in.PatientLang))
		if patientLang != "" && len(patientLang) <= 8 {
			_, updErr := client.UpdateBridgedCallTranslation(ctx, &schedpb.UpdateBridgedCallTranslationRequest{
				SessionId:   sessionID,
				Participant: "patient",
				Translation: &schedpb.BridgedCallTranslationPreferences{
					Enabled:      true,
					LanguageMode: "manual",
					LanguageCode: patientLang,
				},
			})
			if updErr != nil {
				auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "request_staff_transfer", "error", updErr.Error(), correlationID, map[string]any{
					"phase": "set_patient_language",
				})
				// Transfer already succeeded; surface a soft warning rather than failing the whole request.
			}
		}

		session := resp.GetSession()
		payload := map[string]any{
			"session_id":  session.GetSessionId(),
			"status":      session.GetStatus(),
			"hospital_id": session.GetHospitalId(),
			"staff_id":    session.GetStaffId(),
			"message":     "Hospital staff have been notified. Please stay on the line while a clinician joins this call.",
		}
		if patientLang != "" {
			payload["patient_language"] = patientLang
		}
		body, _ := json.Marshal(payload)
		auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "request_staff_transfer", "success", "", correlationID, map[string]any{
			"session_id":  session.GetSessionId(),
			"hospital_id": hospitalID,
			"status":      session.GetStatus(),
		})
		return textResult(string(body)), nil, nil
	})
}
