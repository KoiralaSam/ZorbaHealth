package services

import (
	"context"

	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *PatientService) appendVoiceAudit(
	ctx context.Context,
	eventType string,
	phone string,
	voiceSessionID string,
	metadata map[string]any,
	success bool,
	failureReason string,
) {
	if s.auditClient == nil {
		return
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	if voiceSessionID != "" {
		metadata["voice_session_id"] = voiceSessionID
	}
	metaStruct, err := structpb.NewStruct(metadata)
	if err != nil {
		metaStruct = &structpb.Struct{}
	}
	actorID := "voice:" + voiceSessionID
	if actorID == "voice:" {
		actorID = "voice:unknown"
	}
	_, _ = s.auditClient.AppendAuditEvent(ctx, &auditpb.AppendAuditEventRequest{
		Event: &auditpb.AuditEvent{
			EventId:       uuid.NewString(),
			EventType:     eventType,
			ActorType:     "patient",
			ActorId:       actorID,
			ServiceName:   "patient-service",
			Timestamp:     timestamppb.Now(),
			SuccessStatus: success,
			FailureReason: failureReason,
			Metadata:      metaStruct,
		},
	})
}
