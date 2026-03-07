package rmqconsumer

import (
	"context"
	"encoding/json"
	"log"

	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	messaging "github.com/KoiralaSam/ZorbaHealth/shared/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PatientConsumer struct {
	rabbitmq *messaging.RabbitMQ
}

func NewPatientConsumer(rabbitmq *messaging.RabbitMQ) *PatientConsumer {
	return &PatientConsumer{rabbitmq: rabbitmq}
}

func (c *PatientConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(events.NotifyPatientRegisteredQueue, func(ctx context.Context, message amqp.Delivery) error {
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

		log.Printf("Auth Received a message: %+v", payload)
		return nil
	})
}
