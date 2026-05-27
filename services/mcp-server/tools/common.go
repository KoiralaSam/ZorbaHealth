package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
)

var auditClient auditpb.AuditServiceClient

func ConfigureAuditClient(client auditpb.AuditServiceClient) {
	auditClient = client
}

func verifyToken(token string) (*sharedauth.Claims, error) {
	return sharedauth.VerifyToken(token)
}

func ctxWithForwardedToken(ctx context.Context, token string) context.Context {
	return grpcclient.WithForwardedToken(ctx, token)
}

func auditCompat(db *pgxpool.Pool, claims *sharedauth.Claims, tool, outcome, errMsg string) {
	sharedauth.LogAuditEventAsync(db, claims, tool, outcome, errMsg)
}

func auditStart(ctx context.Context, claims *sharedauth.Claims, eventType, tool string, metadata map[string]any) string {
	correlationID := uuid.NewString()
	appendAuditEvent(ctx, claims, eventType, tool, true, "", correlationID, "start", metadata)
	return correlationID
}

func auditComplete(
	ctx context.Context,
	db *pgxpool.Pool,
	claims *sharedauth.Claims,
	eventType, tool, outcome, errMsg, correlationID string,
	metadata map[string]any,
) {
	appendAuditEvent(ctx, claims, eventType, tool, outcome == "success", errMsg, correlationID, "complete", metadata)
	auditCompat(db, claims, tool, outcome, errMsg)
}

func appendAuditEvent(
	ctx context.Context,
	claims *sharedauth.Claims,
	eventType, tool string,
	success bool,
	failureReason, correlationID, phase string,
	metadata map[string]any,
) {
	if auditClient == nil || claims == nil {
		return
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["phase"] = phase
	metadata["tool_name"] = tool
	metadata["correlation_id"] = correlationID
	metaStruct, err := structpb.NewStruct(metadata)
	if err != nil {
		metaStruct = &structpb.Struct{}
	}
	_, _ = auditClient.AppendAuditEvent(ctx, &auditpb.AppendAuditEventRequest{
		Event: &auditpb.AuditEvent{
			EventId:       uuid.NewString(),
			EventType:     eventType,
			ActorType:     claims.ActorType,
			ActorId:       actorID(claims),
			PatientId:     claims.PatientID,
			ServiceName:   "mcp-server",
			ToolName:      tool,
			Timestamp:     timestamppb.Now(),
			CorrelationId: correlationID,
			SuccessStatus: success,
			FailureReason: failureReason,
			Metadata:      metaStruct,
		},
	})
}

func checkConsent(ctx context.Context, db *pgxpool.Pool, authToken, patientID, consentType, scope string) (bool, string, error) {
	if auditClient != nil {
		if authToken == "" {
			return false, "", fmt.Errorf("missing auth token for consent check")
		}
		auditCtx := ctxWithForwardedToken(ctx, authToken)
		resp, err := auditClient.CheckConsent(auditCtx, &auditpb.CheckConsentRequest{
			PatientId:   patientID,
			ConsentType: consentType,
			Scope:       scope,
		})
		if err != nil {
			return false, "", err
		}
		return resp.GetAllowed(), resp.GetDenialReason(), nil
	}

	switch consentType {
	case sharedaudit.ConsentHealthRecordAccess, sharedaudit.ConsentAISummarization:
		parts := strings.SplitN(scope, ":", 2)
		if len(parts) == 2 && parts[0] == "hospital" && parts[1] != "" {
			allowed, err := sharedauth.CheckConsent(ctx, db, patientID, parts[1])
			if err != nil {
				return false, "", err
			}
			if !allowed {
				return false, "access denied: patient has not consented to share data with your hospital", nil
			}
		}
	}
	return true, "", nil
}

// ensurePatientConsent grants a global-scope consent when missing (e.g. after voice OTP verify).
func ensurePatientConsent(ctx context.Context, authToken, patientID, consentType, source string) {
	if auditClient == nil || patientID == "" || strings.HasPrefix(patientID, "session:") {
		return
	}
	allowed, _, err := checkConsent(ctx, nil, authToken, patientID, consentType, "")
	if err != nil || allowed {
		return
	}
	auditCtx := ctxWithForwardedToken(ctx, authToken)
	_, _ = auditClient.GrantConsent(auditCtx, &auditpb.GrantConsentRequest{
		PatientId:   patientID,
		ConsentType: consentType,
		Scope:       "",
		Source:      source,
	})
}

func actorID(claims *sharedauth.Claims) string {
	switch claims.ActorType {
	case sharedauth.ActorPatient:
		return claims.PatientID
	case sharedauth.ActorStaff:
		return claims.StaffID
	case sharedauth.ActorAdmin:
		return claims.AdminID
	default:
		return ""
	}
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}

func requireToken(token string) error {
	if token == "" {
		return fmt.Errorf("missing _auth")
	}
	return nil
}
