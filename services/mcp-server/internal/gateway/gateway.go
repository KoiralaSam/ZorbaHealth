package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/KoiralaSam/ZorbaHealth/services/mcp-server/internal/registry"
	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
)

type ToolHandler func(ctx context.Context, req *mcp.CallToolRequest, claims *sharedauth.Claims) (*mcp.CallToolResult, any, error)

type Gateway struct {
	db         *pgxpool.Pool
	audit      auditpb.AuditServiceClient
	serviceName string
}

func New(db *pgxpool.Pool, auditClient auditpb.AuditServiceClient, serviceName string) *Gateway {
	return &Gateway{
		db:          db,
		audit:       auditClient,
		serviceName: serviceName,
	}
}

func (g *Gateway) RegisterTool(server *mcp.Server, tool *mcp.Tool, handler ToolHandler) {
	meta, ok := registry.Lookup(tool.Name)
	if !ok {
		meta = registry.ToolMetadata{
			ToolName:      tool.Name,
			Description:   tool.Description,
			AuditRequired: true,
			EventType:     sharedaudit.EventAIToolCalled,
		}
	}

	mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, any, error) {
		authToken, _ := in["_auth"].(string)
		if strings.TrimSpace(authToken) == "" {
			return errorResult("unauthorized"), nil, nil
		}

		claims, err := sharedauth.VerifyToken(authToken)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		if err := g.checkActor(meta, claims); err != nil {
			g.auditCompat(ctx, claims, meta.ToolName, "forbidden", err.Error())
			_ = g.appendAudit(ctx, claims, meta, false, err.Error(), map[string]any{"phase": "start"})
			return errorResult(err.Error()), nil, nil
		}
		if err := g.checkScopes(meta, claims); err != nil {
			g.auditCompat(ctx, claims, meta.ToolName, "forbidden", err.Error())
			_ = g.appendAudit(ctx, claims, meta, false, err.Error(), map[string]any{"phase": "start"})
			return errorResult(err.Error()), nil, nil
		}
		if err := g.checkConsent(ctx, authToken, meta, claims, in); err != nil {
			g.auditCompat(ctx, claims, meta.ToolName, "consent-denied", err.Error())
			_ = g.appendAudit(ctx, claims, meta, false, err.Error(), map[string]any{"phase": "start"})
			return errorResult(err.Error()), nil, nil
		}

		correlationID := uuid.NewString()
		_ = g.appendAudit(ctx, claims, meta, true, "", map[string]any{
			"phase":          "start",
			"correlation_id": correlationID,
		})

		result, out, err := handler(ctx, req, claims)
		if err != nil {
			g.auditCompat(ctx, claims, meta.ToolName, "error", err.Error())
			_ = g.appendAudit(ctx, claims, meta, false, err.Error(), map[string]any{
				"phase":          "complete",
				"correlation_id": correlationID,
			})
			return errorResult(err.Error()), nil, nil
		}

		isError := result != nil && result.IsError
		failureReason := ""
		if isError {
			failureReason = contentText(result)
			g.auditCompat(ctx, claims, meta.ToolName, "error", failureReason)
		} else {
			g.auditCompat(ctx, claims, meta.ToolName, "success", "")
		}
		_ = g.appendAudit(ctx, claims, meta, !isError, failureReason, map[string]any{
			"phase":          "complete",
			"correlation_id": correlationID,
		})
		return result, out, nil
	})
}

func (g *Gateway) checkActor(meta registry.ToolMetadata, claims *sharedauth.Claims) error {
	if len(meta.AllowedActorTypes) == 0 {
		return nil
	}
	for _, actorType := range meta.AllowedActorTypes {
		if claims.ActorType == actorType {
			return nil
		}
	}
	return fmt.Errorf("forbidden: unsupported actor type")
}

func (g *Gateway) checkScopes(meta registry.ToolMetadata, claims *sharedauth.Claims) error {
	for _, permission := range meta.RequiredPermissions {
		if !sharedauth.HasScope(claims, permission) {
			return fmt.Errorf("forbidden: missing %s", permission)
		}
	}
	return nil
}

func (g *Gateway) checkConsent(ctx context.Context, authToken string, meta registry.ToolMetadata, claims *sharedauth.Claims, in map[string]any) error {
	if !meta.RequiresPatientConsent {
		return nil
	}
	patientID := claims.PatientID
	if patientID == "" {
		if value, ok := in["patientID"].(string); ok {
			patientID = value
		}
	}
	if patientID == "" {
		return fmt.Errorf("consent check failed: patient id is required")
	}
	if g.audit == nil {
		return nil
	}
	auditCtx := grpcclient.WithForwardedToken(ctx, authToken)
	resp, err := g.audit.CheckConsent(auditCtx, &auditpb.CheckConsentRequest{
		PatientId:   patientID,
		ConsentType: meta.ConsentType,
		Scope:       scopeFromInput(in),
	}, grpc.WaitForReady(true))
	if err != nil {
		return fmt.Errorf("consent check failed")
	}
	if !resp.GetAllowed() {
		if resp.GetDenialReason() != "" {
			return fmt.Errorf("%s", resp.GetDenialReason())
		}
		return fmt.Errorf("access denied: patient consent not granted")
	}
	return nil
}

func (g *Gateway) appendAudit(ctx context.Context, claims *sharedauth.Claims, meta registry.ToolMetadata, success bool, failureReason string, metadata map[string]any) error {
	if g.audit == nil || !meta.AuditRequired {
		return nil
	}
	s, err := structpb.NewStruct(metadata)
	if err != nil {
		s = &structpb.Struct{}
	}
	_, err = g.audit.AppendAuditEvent(ctx, &auditpb.AppendAuditEventRequest{
		Event: &auditpb.AuditEvent{
			EventId:       uuid.NewString(),
			EventType:     meta.EventType,
			ActorType:     claims.ActorType,
			ActorId:       actorID(claims),
			PatientId:     claims.PatientID,
			ServiceName:   g.serviceName,
			Timestamp:     timestamppb.Now(),
			ToolName:      meta.ToolName,
			SuccessStatus: success,
			FailureReason: failureReason,
			Metadata:      s,
		},
	}, grpc.WaitForReady(true))
	return err
}

func (g *Gateway) auditCompat(ctx context.Context, claims *sharedauth.Claims, tool, outcome, errMsg string) {
	if g.db != nil {
		sharedauth.LogAuditEventAsync(g.db, claims, tool, outcome, errMsg)
	}
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

func scopeFromInput(in map[string]any) string {
	if sessionID, ok := in["sessionID"].(string); ok {
		return sessionID
	}
	if patientID, ok := in["patientID"].(string); ok {
		return patientID
	}
	return ""
}

func contentText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	var parts []string
	for _, item := range result.Content {
		if text, ok := item.(*mcp.TextContent); ok && text.Text != "" {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}
