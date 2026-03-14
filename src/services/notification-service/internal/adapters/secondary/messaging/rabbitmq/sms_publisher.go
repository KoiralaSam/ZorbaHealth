package rabbitmq

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
)

type SMSPublisher struct {
	rabbitmq *messaging.RabbitMQ
}

func NewSMSPublisher(rmq *messaging.RabbitMQ) *SMSPublisher {
	return &SMSPublisher{rabbitmq: rmq}
}

func (p *SMSPublisher) SendSMS(ctx context.Context) error {
	// payload := events.PatientEventData{

	// }
	// jsonData, err := json.Marshal(payload)
	// if err != nil {
	// 	return err
	// }
	// return p.rabbitmq.PublishMessage(ctx, messaging.PatientExchange, contracts.PatientEventRegistered, contracts.AmqpMessage{
	// 	OwnerID: patient.ID.String(),
	// 	Data:    jsonData,
	// }
	return nil
}
