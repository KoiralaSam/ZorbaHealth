package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	patientportalpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patientportal"
)

type updateWelfareRunStatusInput struct {
	PatientID string `json:"patientID" jsonschema:"verified patient id"`
	RunID     string `json:"runID" jsonschema:"welfare check run id"`
	Status    string `json:"status" jsonschema:"answered, completed, missed, or failed"`
	Reason    string `json:"reason" jsonschema:"optional failure or miss reason"`
	Auth      string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterUpdateWelfareRunStatus(s *mcp.Server, db *pgxpool.Pool, client patientportalpb.PatientPortalServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_welfare_run_status",
		Description: "Update lifecycle status for a scheduled welfare-check outbound call run",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in updateWelfareRunStatusInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		patientID := strings.TrimSpace(in.PatientID)
		runID := strings.TrimSpace(in.RunID)
		status := strings.ToLower(strings.TrimSpace(in.Status))
		if patientID == "" || runID == "" {
			return errorResult("patientID and runID are required"), nil, nil
		}
		switch status {
		case "answered", "completed", "missed", "failed":
		default:
			return errorResult("status must be answered, completed, missed, or failed"), nil, nil
		}
		if claims.PatientID != "" && claims.PatientID != patientID {
			return errorResult("patient_id mismatch"), nil, nil
		}

		correlationID := auditStart(ctx, claims, "AI_TOOL_CALLED", "update_welfare_run_status", map[string]any{
			"run_id":  runID,
			"status":  status,
			"patient": patientID,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.UpdateWelfareRunLifecycle(ctx, &patientportalpb.UpdateWelfareRunLifecycleRequest{
			PatientId: patientID,
			RunId:     runID,
			Status:    status,
			Reason:    strings.TrimSpace(in.Reason),
		})
		if err != nil {
			auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "update_welfare_run_status", "error", err.Error(), correlationID, nil)
			return errorResult("failed to update welfare run status"), nil, nil
		}
		body, _ := json.Marshal(map[string]any{
			"runID":    resp.GetRun().GetId(),
			"status":   resp.GetRun().GetStatus(),
			"attempts": resp.GetRun().GetAttempts(),
		})
		auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "update_welfare_run_status", "success", "", correlationID, map[string]any{
			"status": status,
		})
		return textResult(string(body)), nil, nil
	})
}
