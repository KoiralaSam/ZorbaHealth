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
	ReceiveSMS(ctx context.Context, phoneNumber, message string) error
}

