package inbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/shared/events"
)

// NotificationService is the inbound port implemented by the core notification service.
// Primary adapters (RabbitMQ consumers, HTTP webhook handlers, etc.) should depend on this interface.
type NotificationService interface {
	SendPendingVerificationEmail(ctx context.Context, req *events.PendingRegistrationData, token string) error
	SendOTP(ctx context.Context, phone string, otp string) error
	SendEmergencyEscalationSMS(ctx context.Context, phone string, reason string) error
	SendEmergencyEscalationAlerts(ctx context.Context, phones []string, reason string) []error
	ReceiveSMS(ctx context.Context, phoneNumber, message string) error
	SendMeetingRequestedNotifications(ctx context.Context, data *events.MeetingRequestedData) error
	SendMeetingScheduledNotifications(ctx context.Context, data *events.MeetingScheduledData) error
}
