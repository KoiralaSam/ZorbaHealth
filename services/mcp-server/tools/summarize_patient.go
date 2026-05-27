package tools

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	healthpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
)

type summarizePatientRecordInput struct {
	PatientID string `json:"patientID" jsonschema:"target patient ID"`
	Focus     string `json:"focus,omitempty" jsonschema:"summary focus"`
	Auth      string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterSummarizePatientRecord(s *mcp.Server, db *pgxpool.Pool, client healthpb.HealthRecordServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "summarize_patient_record",
		Description: "Summarize a patient's records for hospital staff",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in summarizePatientRecordInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		if err := sharedauth.RequireActorType(claims, sharedauth.ActorStaff); err != nil {
			auditCompat(db, claims, "summarize_patient_record", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		if !sharedauth.HasScope(claims, "patient:read") {
			auditCompat(db, claims, "summarize_patient_record", "forbidden", "missing patient:read")
			return errorResult("forbidden: missing patient:read"), nil, nil
		}

		ok, denialReason, err := checkConsent(ctx, db, in.Auth, in.PatientID, sharedaudit.ConsentAISummarization, "hospital:"+claims.HospitalID)
		if err != nil {
			auditCompat(db, claims, "summarize_patient_record", "error", err.Error())
			return errorResult("consent check failed"), nil, nil
		}
		if !ok {
			if denialReason == "" {
				denialReason = "access denied: patient has not consented to share data with your hospital"
			}
			auditCompat(db, claims, "summarize_patient_record", "consent-denied", denialReason)
			return errorResult(denialReason), nil, nil
		}

		focus := in.Focus
		if focus == "" {
			focus = "full"
		}

		correlationID := auditStart(ctx, claims, sharedaudit.EventHealthRecordSummarized, "summarize_patient_record", map[string]any{
			"patient_id": in.PatientID,
			"focus":      focus,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.SummarizeRecords(ctx, &healthpb.SummarizeRequest{
			PatientId:  in.PatientID,
			HospitalId: claims.HospitalID,
			Focus:      focus,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventHealthRecordSummarized, "summarize_patient_record", "error", err.Error(), correlationID, nil)
			return errorResult("summarization failed"), nil, nil
		}

		auditComplete(ctx, db, claims, sharedaudit.EventHealthRecordSummarized, "summarize_patient_record", "success", "", correlationID, nil)
		return textResult(resp.GetSummary()), nil, nil
	})
}
