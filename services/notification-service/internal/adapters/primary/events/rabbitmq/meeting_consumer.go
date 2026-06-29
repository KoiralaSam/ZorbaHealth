package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/logging"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

type MeetingConsumer struct {
	rabbitmq *messaging.RabbitMQ
	svc      inbound.NotificationService
}

func NewMeetingConsumer(rabbitmq *messaging.RabbitMQ, svc inbound.NotificationService) *MeetingConsumer {
	return &MeetingConsumer{rabbitmq: rabbitmq, svc: svc}
}

func (c *MeetingConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(events.NotifyMeetingScheduledQueue, func(ctx context.Context, message amqp.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(message.Body, &envelope); err != nil {
			logging.Error("meeting scheduled event decode failed", err)
			return err
		}
		var payload events.MeetingScheduledData
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			logging.Error("meeting scheduled payload decode failed", err)
			return err
		}
		if err := c.svc.SendMeetingScheduledNotifications(ctx, &payload); err != nil {
			logging.Error("meeting scheduled notifications failed", err)
		}
		return nil
	})
}

func (c *MeetingConsumer) ListenRequested() error {
	return c.rabbitmq.ConsumeMessages(events.NotifyMeetingRequestedQueue, func(ctx context.Context, message amqp.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(message.Body, &envelope); err != nil {
			logging.Error("meeting requested event decode failed", err)
			return err
		}
		var payload events.MeetingRequestedData
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			logging.Error("meeting requested payload decode failed", err)
			return err
		}
		if err := c.svc.SendMeetingRequestedNotifications(ctx, &payload); err != nil {
			logging.Error("meeting requested notifications failed", err)
		}
		return nil
	})
}
