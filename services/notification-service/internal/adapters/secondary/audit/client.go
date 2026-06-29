package audit

import (
	"context"
	"os"

	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/outbound"
	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	client auditpb.AuditServiceClient
}

func NewClient(client auditpb.AuditServiceClient) outbound.NotificationAudit {
	if client == nil {
		return noop{}
	}
	return &Client{client: client}
}

type noop struct{}

func (noop) RecordNotificationSent(context.Context, string, string, string, string, string, string, bool, string) error {
	return nil
}

func (c *Client) RecordNotificationSent(ctx context.Context, patientID, correlationID, meetingID, channel, recipientRole, template string, success bool, failureReason string) error {
	meta, err := structpb.NewStruct(map[string]any{
		"meeting_id":     meetingID,
		"channel":        channel,
		"recipient_role": recipientRole,
		"template":       template,
		"correlation_id": correlationID,
	})
	if err != nil {
		meta = &structpb.Struct{}
	}
	ctx = withInternalMetadata(ctx)
	_, err = c.client.AppendAuditEvent(ctx, &auditpb.AppendAuditEventRequest{
		Event: &auditpb.AuditEvent{
			EventId:       uuid.NewString(),
			EventType:     sharedaudit.EventNotificationSent,
			ActorType:     "system",
			ActorId:       "notification-service",
			PatientId:     patientID,
			ServiceName:   "notification-service",
			Timestamp:     timestamppb.Now(),
			CorrelationId: correlationID,
			SuccessStatus: success,
			FailureReason: failureReason,
			Metadata:      meta,
		},
	})
	return err
}

func withInternalMetadata(ctx context.Context) context.Context {
	secret := os.Getenv("INTERNAL_SERVICE_SECRET")
	if secret == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "x-internal-token", secret)
}
