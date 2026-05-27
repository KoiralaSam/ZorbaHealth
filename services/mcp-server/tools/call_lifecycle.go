package tools

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
)

type notifyCallLifecycleInput struct {
	EventType string `json:"eventType" jsonschema:"call.started or call.ended"`
	SessionID string `json:"sessionID" jsonschema:"LiveKit room/session id"`
	PatientID string `json:"patientID" jsonschema:"verified patient id"`
	Auth      string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterNotifyCallLifecycle(s *mcp.Server, db *pgxpool.Pool, callsRMQ *messaging.RabbitMQ) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "notify_call_lifecycle",
		Description: "Publish call lifecycle events so the patient app can start or stop location sharing",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in notifyCallLifecycleInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		if in.SessionID == "" || in.PatientID == "" {
			return errorResult("sessionID and patientID are required"), nil, nil
		}
		routingKey := in.EventType
		switch routingKey {
		case contracts.CallEventStarted, contracts.CallEventEnded:
		default:
			return errorResult("eventType must be call.started or call.ended"), nil, nil
		}
		if claims.SessionID != "" && claims.SessionID != in.SessionID {
			return errorResult("session_id mismatch"), nil, nil
		}

		correlationID := auditStart(ctx, claims, "AI_TOOL_CALLED", "notify_call_lifecycle", map[string]any{
			"event_type": routingKey,
			"session_id": in.SessionID,
		})

		if callsRMQ == nil {
			auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "notify_call_lifecycle", "error", "calls rabbitmq not configured", correlationID, nil)
			return errorResult("call events unavailable"), nil, nil
		}

		payload, err := json.Marshal(events.CallEvent{
			EventType: routingKey,
			PatientID: in.PatientID,
			SessionID: in.SessionID,
		})
		if err != nil {
			auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "notify_call_lifecycle", "error", err.Error(), correlationID, nil)
			return errorResult("failed to marshal call event"), nil, nil
		}

		err = callsRMQ.Publish(ctx, events.CallsExchange, routingKey, amqp.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			DeliveryMode: amqp.Persistent,
		})
		if err != nil {
			auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "notify_call_lifecycle", "error", err.Error(), correlationID, nil)
			return errorResult("failed to publish call event"), nil, nil
		}

		auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "notify_call_lifecycle", "success", "", correlationID, nil)
		return textResult("Call lifecycle event published."), nil, nil
	})
}
