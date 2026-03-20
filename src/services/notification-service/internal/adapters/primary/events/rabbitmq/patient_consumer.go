package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
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
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		var payload events.PatientEventData
		if err := json.Unmarshal(PatientEvent.Data, &payload); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		if payload.RegisterRequest == nil {
			return fmt.Errorf("pending verification event missing register_request")
		}

		// IMPORTANT: Don't return provider errors (email/SMS) here, otherwise RabbitMQ will redeliver
		// and we can get stuck in an infinite retry loop (e.g., SendGrid credits exceeded).
		// We only return errors for malformed/unprocessable messages.

		if err := c.svc.SendPendingVerificationEmail(ctx, payload.RegisterRequest, PatientEvent.OwnerID); err != nil {
			log.Printf("Failed to send verification email: %v", err)
		} else {
			log.Printf("Sent verification email to %s", payload.RegisterRequest.Email)
		}

		if payload.RegisterRequest.PhoneNumber != "" && payload.RegisterRequest.Otp != "" {
			if err := c.svc.SendOTP(ctx, payload.RegisterRequest.PhoneNumber, payload.RegisterRequest.Otp); err != nil {
				log.Printf("Failed to send OTP SMS: %v", err)
			} else {
				log.Printf("Sent OTP to %s", payload.RegisterRequest.PhoneNumber)
			}
		}

		// Ack the message regardless of provider outcome to prevent infinite redelivery.
		return nil
	})
}
