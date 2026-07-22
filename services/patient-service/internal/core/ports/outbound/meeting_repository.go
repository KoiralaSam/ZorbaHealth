package outbound

import (
	"context"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/google/uuid"
)

type MeetingRepository interface {
	Insert(ctx context.Context, meeting *models.ScheduledMeeting) (*models.ScheduledMeeting, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.ScheduledMeeting, error)
	List(ctx context.Context, filter models.ListMeetingsFilter) ([]models.ScheduledMeeting, error)
	MarkScheduled(ctx context.Context, meeting *models.ScheduledMeeting) (*models.ScheduledMeeting, error)
	Cancel(ctx context.Context, id uuid.UUID) (*models.ScheduledMeeting, error)
	ClaimDueMeetingReminders(ctx context.Context, within time.Duration, limit int32) ([]models.ScheduledMeeting, error)
	MarkMeetingReminderSent(ctx context.Context, meeting *models.ScheduledMeeting) (*models.ScheduledMeeting, error)
	HasActiveConsent(ctx context.Context, patientID, hospitalID uuid.UUID) (bool, error)
	GetStaffByID(ctx context.Context, staffID uuid.UUID) (*models.StaffSummary, error)
	ListSchedulableStaff(ctx context.Context, hospitalID uuid.UUID) ([]models.StaffSummary, error)
}

type SchedulingPublisher interface {
	PublishMeetingRequested(ctx context.Context, data *events.MeetingRequestedData) error
	PublishMeetingScheduled(ctx context.Context, data *events.MeetingScheduledData) error
	PublishMeetingReminder(ctx context.Context, data *events.MeetingReminderData) error
}
