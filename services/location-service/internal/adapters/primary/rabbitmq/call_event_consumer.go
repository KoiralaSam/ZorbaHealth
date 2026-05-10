package rabbitmq

import (
	"context"
	"encoding/json"
	"log"

	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	messaging "github.com/KoiralaSam/ZorbaHealth/shared/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

type CallEventConsumer struct {
	rabbitmq *messaging.RabbitMQ
	svc      inbound.LocationService
}

func NewCallEventConsumer(rabbitmq *messaging.RabbitMQ, svc inbound.LocationService) *CallEventConsumer {
	return &CallEventConsumer{rabbitmq: rabbitmq, svc: svc}
}

func (c *CallEventConsumer) Listen() error {
	log.Println("call event consumer started")
	return c.rabbitmq.ConsumeMessages(events.LocationCallEventsQueue, func(ctx context.Context, message amqp.Delivery) error {
		var event events.CallEvent
		if err := json.Unmarshal(message.Body, &event); err != nil {
			log.Printf("call event unmarshal error: %v", err)
			return err
		}

		if err := c.svc.HandleCallEvent(ctx, event); err != nil {
			log.Printf("call event handle error: %v", err)
			return err
		}

		return nil
	})
}
