package rabbitmq

import (
	"context"
	"encoding/json"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	sharedlogging "github.com/KoiralaSam/ZorbaHealth/shared/logging"
	messaging "github.com/KoiralaSam/ZorbaHealth/shared/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PatientConsumer struct {
	rabbitmq *messaging.RabbitMQ
	svc      inbound.NotificationService
}

func NewPatientConsumer(rabbitmq *messaging.RabbitMQ, svc inbound.NotificationService) *PatientConsumer {
	return &PatientConsumer{rabbitmq: rabbitmq, svc: svc}
}

func (c *PatientConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(events.NotifyPatientPendingVerificationQueue, func(ctx context.Context, message amqp.Delivery) error {
		var PatientEvent contracts.AmqpMessage
		if err := json.Unmarshal(message.Body, &PatientEvent); err != nil {
			sharedlogging.Error("notification patient event decode failed", err)
			return err
		}

		var payload events.PatientEventData
		if err := json.Unmarshal(PatientEvent.Data, &payload); err != nil {
			sharedlogging.Error("notification patient payload decode failed", err)
			return err
		}

		// IMPORTANT: Don't return provider errors (email/SMS) here, otherwise RabbitMQ will redeliver
		// and we can get stuck in an infinite retry loop (e.g., provider quota exceeded).
		// We only return errors for malformed/unprocessable messages.

		if payload.RegisterRequest != nil {
			if err := c.svc.SendPendingVerificationEmail(ctx, payload.RegisterRequest, PatientEvent.OwnerID); err != nil {
				sharedlogging.Error("notification email send failed", err)
			} else {
				sharedlogging.Info("notification email sent",
					"email_hash", sharedlogging.HashIdentifier(payload.RegisterRequest.Email),
				)
			}
		}

		if payload.RegisterRequest != nil && payload.RegisterRequest.PhoneNumber != "" && payload.RegisterRequest.Otp != "" {
			if err := c.svc.SendOTP(ctx, payload.RegisterRequest.PhoneNumber, payload.RegisterRequest.Otp); err != nil {
				sharedlogging.Error("notification otp send failed", err)
			} else {
				sharedlogging.Info("notification otp sent",
					"phone_hash", sharedlogging.HashIdentifier(payload.RegisterRequest.PhoneNumber),
				)
			}
		}

		if payload.PhoneVerification != nil && payload.PhoneVerification.PhoneNumber != "" && payload.PhoneVerification.Otp != "" {
			if err := c.svc.SendOTP(ctx, payload.PhoneVerification.PhoneNumber, payload.PhoneVerification.Otp); err != nil {
				sharedlogging.Error("notification phone verification otp send failed", err)
			} else {
				sharedlogging.Info("notification phone verification otp sent",
					"phone_hash", sharedlogging.HashIdentifier(payload.PhoneVerification.PhoneNumber),
				)
			}
		}

		if payload.RegisterRequest == nil && payload.PhoneVerification == nil {
			return domainErrors.ErrPendingVerificationEventMissingRegisterRequest
		}

		// Ack the message regardless of provider outcome to prevent infinite redelivery.
		return nil
	})
}
