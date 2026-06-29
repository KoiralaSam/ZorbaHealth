package rabbitmq

import (
	"context"
	"encoding/json"

	outbound "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
)

type SchedulingPublisher struct {
	rabbitmq *messaging.RabbitMQ
}

func NewSchedulingPublisher(rmq *messaging.RabbitMQ) outbound.SchedulingPublisher {
	return &SchedulingPublisher{rabbitmq: rmq}
}

func (p *SchedulingPublisher) PublishMeetingRequested(ctx context.Context, data *events.MeetingRequestedData) error {
	if data == nil {
		return nil
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return p.rabbitmq.PublishMessage(ctx, events.PatientExchange, contracts.PatientEventMeetingRequested, contracts.AmqpMessage{
		OwnerID: data.PatientID,
		Data:    jsonData,
	})
}

func (p *SchedulingPublisher) PublishMeetingScheduled(ctx context.Context, data *events.MeetingScheduledData) error {
	if data == nil {
		return nil
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	owner := data.PatientID
	return p.rabbitmq.PublishMessage(ctx, events.PatientExchange, contracts.PatientEventMeetingScheduled, contracts.AmqpMessage{
		OwnerID: owner,
		Data:    jsonData,
	})
}
