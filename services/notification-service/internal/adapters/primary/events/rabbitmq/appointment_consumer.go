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

type AppointmentConsumer struct {
	rabbitmq *messaging.RabbitMQ
	svc      inbound.NotificationService
}

func NewAppointmentConsumer(rabbitmq *messaging.RabbitMQ, svc inbound.NotificationService) *AppointmentConsumer {
	return &AppointmentConsumer{rabbitmq: rabbitmq, svc: svc}
}

func (c *AppointmentConsumer) ListenBooked() error {
	return c.rabbitmq.ConsumeMessages(events.NotifyAppointmentBookedQueue, func(ctx context.Context, message amqp.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(message.Body, &envelope); err != nil {
			logging.Error("appointment booked event decode failed", err)
			return err
		}
		var payload events.AppointmentBookedData
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			logging.Error("appointment booked payload decode failed", err)
			return err
		}
		if err := c.svc.SendAppointmentBookedNotifications(ctx, &payload); err != nil {
			logging.Error("appointment booked notifications failed", err)
		}
		return nil
	})
}

func (c *AppointmentConsumer) ListenCancelled() error {
	return c.rabbitmq.ConsumeMessages(events.NotifyAppointmentCancelledQueue, func(ctx context.Context, message amqp.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(message.Body, &envelope); err != nil {
			logging.Error("appointment cancelled event decode failed", err)
			return err
		}
		var payload events.AppointmentCancelledData
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			logging.Error("appointment cancelled payload decode failed", err)
			return err
		}
		if err := c.svc.SendAppointmentCancelledNotifications(ctx, &payload); err != nil {
			logging.Error("appointment cancelled notifications failed", err)
		}
		return nil
	})
}
