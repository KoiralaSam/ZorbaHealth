package inbound

import (
	"context"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
)

type SchedulingService interface {
	ScheduleHealthStaffMeeting(ctx context.Context, cmd *models.ScheduleMeetingCommand) (*models.ScheduledMeeting, error)
	AcceptScheduledMeeting(ctx context.Context, meetingID string, actor models.ScheduleActor) (*models.ScheduledMeeting, error)
	RescheduleScheduledMeeting(ctx context.Context, meetingID string, actor models.ScheduleActor, startsAt time.Time, durationMinutes int32, timezone, title string) (*models.ScheduledMeeting, error)
	ListScheduledMeetings(ctx context.Context, filter models.ListMeetingsFilter) ([]models.ScheduledMeeting, error)
	CancelScheduledMeeting(ctx context.Context, meetingID string, actor models.ScheduleActor, reason string) (*models.ScheduledMeeting, error)
	ListSchedulableStaff(ctx context.Context, patientID, hospitalID string) ([]models.StaffSummary, error)
	RequestBridgedCallTransfer(ctx context.Context, cmd *models.RequestBridgedCallTransferCommand) (*models.BridgedCallSession, error)
	ConnectBridgedCall(ctx context.Context, sessionID string, actor models.ScheduleActor, participantIdentity string, accessToken string) (*models.BridgedCallConnectResult, error)
	GetBridgedCallSession(ctx context.Context, sessionID string, actor models.ScheduleActor) (*models.BridgedCallSession, error)
	MintBridgedCallPatientToken(ctx context.Context, session *models.BridgedCallSession) (*outbound.LiveKitRoomToken, error)
	ListBridgedCallSessions(ctx context.Context, actor models.ScheduleActor, status string, limit int) ([]*models.BridgedCallSession, error)
	UpdateBridgedCallTranslation(ctx context.Context, cmd *models.UpdateBridgedCallTranslationCommand) (*models.BridgedCallSession, error)
	EndBridgedCall(ctx context.Context, sessionID string, actor models.ScheduleActor, reason string) (*models.BridgedCallSession, error)
}
