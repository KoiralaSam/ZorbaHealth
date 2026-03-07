package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	messaging "github.com/KoiralaSam/ZorbaHealth/shared/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PatientConsumer struct {
	rabbitmq *messaging.RabbitMQ
	svc      *services.NotificationService
}

func NewPatientConsumer(rabbitmq *messaging.RabbitMQ, svc *services.NotificationService) *PatientConsumer {
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

		if err := c.svc.SendPendingVerificationEmail(ctx, payload.RegisterRequest, PatientEvent.OwnerID); err != nil {
			log.Printf("Failed to send verification email: %v", err)
			return err
		}

		log.Printf("Sent verification email to %s", payload.RegisterRequest.Email)
		return nil
	})
}
