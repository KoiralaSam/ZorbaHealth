package audit

import (
	"context"
	"log/slog"

	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/ports/outbound"
	"github.com/google/uuid"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Logger struct {
	client auditpb.AuditServiceClient
}

func NewLogger(client auditpb.AuditServiceClient) outbound.AuditLogger {
	return &Logger{client: client}
}

func (l *Logger) Log(ctx context.Context, eventType, outcome, correlationID string, attrs map[string]any) {
	if l == nil || l.client == nil {
		return
	}
	safe := make(map[string]any, len(attrs))
	for k, v := range attrs {
		safe[k] = v
	}
	meta, err := structpb.NewStruct(safe)
	if err != nil {
		meta = &structpb.Struct{}
	}
	success := outcome == "success"
	failure := ""
	if !success {
		failure = outcome
		if r, ok := attrs["reason"].(string); ok {
			failure = r
		}
	}
	_, err = l.client.AppendAuditEvent(ctx, &auditpb.AppendAuditEventRequest{
		Event: &auditpb.AuditEvent{
			EventId:       uuid.NewString(),
			EventType:     eventType,
			ServiceName:   "appointment-service",
			Timestamp:     timestamppb.Now(),
			CorrelationId: correlationID,
			SuccessStatus: success,
			FailureReason: failure,
			Metadata:      meta,
		},
	})
	if err != nil {
		slog.Warn("appointment audit append failed", "event_type", eventType, "error", err.Error())
	}
}
