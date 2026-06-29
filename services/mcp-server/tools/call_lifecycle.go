package tools

import (
	"context"
	"encoding/json"
	"strings"
	"time"

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

		if err := persistCallLifecycle(ctx, db, in.PatientID, in.SessionID, routingKey); err != nil {
			auditComplete(ctx, db, claims, "AI_TOOL_CALLED", "notify_call_lifecycle", "error", err.Error(), correlationID, nil)
			return errorResult("failed to persist call lifecycle"), nil, nil
		}

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

func persistCallLifecycle(ctx context.Context, db *pgxpool.Pool, patientID, sessionID, eventType string) error {
	if db == nil {
		return nil
	}
	patientID = strings.TrimSpace(patientID)
	sessionID = strings.TrimSpace(sessionID)
	if patientID == "" || sessionID == "" {
		return nil
	}

	now := time.Now().UTC()
	switch eventType {
	case contracts.CallEventStarted:
		_, err := db.Exec(ctx, `
			INSERT INTO calls (patient_id, livekit_room_id, status, started_at, ended_at)
			VALUES ($1::uuid, $2, 'active', $3, NULL)
			ON CONFLICT (livekit_room_id) DO UPDATE
			SET patient_id = EXCLUDED.patient_id,
			    status = 'active',
			    started_at = COALESCE(calls.started_at, EXCLUDED.started_at),
			    ended_at = NULL
		`, patientID, sessionID, now)
		return err
	case contracts.CallEventEnded:
		_, err := db.Exec(ctx, `
			INSERT INTO calls (patient_id, livekit_room_id, status, started_at, ended_at)
			VALUES ($1::uuid, $2, 'ended', $3, $3)
			ON CONFLICT (livekit_room_id) DO UPDATE
			SET patient_id = EXCLUDED.patient_id,
			    status = 'ended',
			    started_at = COALESCE(calls.started_at, EXCLUDED.started_at),
			    ended_at = EXCLUDED.ended_at
		`, patientID, sessionID, now)
		return err
	default:
		return nil
	}
}
