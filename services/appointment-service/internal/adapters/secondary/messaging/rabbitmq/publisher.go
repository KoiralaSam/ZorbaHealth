package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
)

type AppointmentPublisher struct {
	rabbitmq *messaging.RabbitMQ
}

func NewAppointmentPublisher(rmq *messaging.RabbitMQ) outbound.EventPublisher {
	return &AppointmentPublisher{rabbitmq: rmq}
}

func (p *AppointmentPublisher) PublishAppointmentBooked(ctx context.Context, data *events.AppointmentBookedData) error {
	if data == nil {
		return nil
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return p.rabbitmq.PublishMessage(ctx, events.PatientExchange, contracts.AppointmentEventBooked, contracts.AmqpMessage{
		OwnerID: data.PatientID,
		Data:    jsonData,
	})
}

func (p *AppointmentPublisher) PublishAppointmentCancelled(ctx context.Context, data *events.AppointmentCancelledData) error {
	if data == nil {
		return nil
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return p.rabbitmq.PublishMessage(ctx, events.PatientExchange, contracts.AppointmentEventCancelled, contracts.AmqpMessage{
		OwnerID: data.PatientID,
		Data:    jsonData,
	})
}
