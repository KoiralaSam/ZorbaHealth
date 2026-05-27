package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
)

type logEscalationInput struct {
	SessionID         string   `json:"sessionID" jsonschema:"voice session ID"`
	PatientID         string   `json:"patientID,omitempty" jsonschema:"optional patient ID"`
	CallerPhone       string   `json:"callerPhone,omitempty" jsonschema:"optional caller phone"`
	Reason            string   `json:"reason" jsonschema:"escalation reason"`
	Severity          string   `json:"severity,omitempty" jsonschema:"low, medium, or high"`
	TransferRequested bool     `json:"transferRequested,omitempty" jsonschema:"request designated emergency transfer"`
	TransferTarget    string   `json:"transferTarget,omitempty" jsonschema:"designated emergency transfer number"`
	AlertPhoneNumbers []string `json:"alertPhoneNumbers,omitempty" jsonschema:"additional phones to notify"`
	TranscriptExcerpt string   `json:"transcriptExcerpt,omitempty" jsonschema:"short transcript excerpt for operators"`
	Auth              string   `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterLogEscalation(s *mcp.Server, db *pgxpool.Pool, rabbitmq *messaging.RabbitMQ) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "log_escalation",
		Description: "Record an emergency escalation and notify downstream systems",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in logEscalationInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		if in.Reason == "" {
			auditCompat(db, claims, "log_escalation", "forbidden", "reason is required")
			return errorResult("reason is required"), nil, nil
		}

		correlationID := auditStart(ctx, claims, sharedaudit.EventEmergencyEscalationTriggered, "log_escalation", map[string]any{
			"session_id":         in.SessionID,
			"severity":           in.Severity,
			"transfer_requested": in.TransferRequested,
			"transfer_target":    in.TransferTarget,
		})

		payload, err := json.Marshal(events.EmergencyEscalationData{
			SessionID:         in.SessionID,
			PatientID:         in.PatientID,
			CallerPhone:       in.CallerPhone,
			Reason:            in.Reason,
			Severity:          in.Severity,
			TransferRequested: in.TransferRequested,
			TransferTarget:    strings.TrimSpace(in.TransferTarget),
			AlertPhoneNumbers: sanitizePhoneTargets(in.AlertPhoneNumbers),
			TranscriptExcerpt: strings.TrimSpace(in.TranscriptExcerpt),
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventEmergencyEscalationTriggered, "log_escalation", "error", err.Error(), correlationID, nil)
			return errorResult("failed to marshal escalation"), nil, nil
		}
		if rabbitmq != nil {
			err = rabbitmq.PublishMessage(ctx, events.EscalationExchange, contracts.EmergencyEscalatedEvent, contracts.AmqpMessage{
				OwnerID: claims.PatientID,
				Data:    payload,
			})
			if err != nil {
				auditComplete(ctx, db, claims, sharedaudit.EventEmergencyEscalationTriggered, "log_escalation", "error", err.Error(), correlationID, nil)
				return errorResult("failed to dispatch escalation"), nil, nil
			}
		}

		auditComplete(ctx, db, claims, sharedaudit.EventEmergencyEscalationTriggered, "log_escalation", "success", "", correlationID, map[string]any{
			"session_id":         in.SessionID,
			"severity":           in.Severity,
			"transfer_requested": in.TransferRequested,
			"transfer_target":    strings.TrimSpace(in.TransferTarget),
		})
		return textResult("Emergency escalation recorded."), nil, nil
	})
}

func sanitizePhoneTargets(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
