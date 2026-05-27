package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	sharedlogging "github.com/KoiralaSam/ZorbaHealth/shared/logging"
	messaging "github.com/KoiralaSam/ZorbaHealth/shared/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

type EmergencyConsumer struct {
	rabbitmq *messaging.RabbitMQ
	svc      inbound.NotificationService
}

func NewEmergencyConsumer(rabbitmq *messaging.RabbitMQ, svc inbound.NotificationService) *EmergencyConsumer {
	return &EmergencyConsumer{rabbitmq: rabbitmq, svc: svc}
}

func (c *EmergencyConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(events.NotifyEmergencyEscalationQueue, func(ctx context.Context, message amqp.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(message.Body, &envelope); err != nil {
			sharedlogging.Error("notification emergency event decode failed", err)
			return err
		}

		var payload events.EmergencyEscalationData
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			sharedlogging.Error("notification emergency payload decode failed", err)
			return err
		}

		if payload.CallerPhone != "" {
			if err := c.svc.SendEmergencyEscalationSMS(ctx, payload.CallerPhone, payload.Reason); err != nil {
				sharedlogging.Error("notification emergency escalation sms failed", err,
					"session_id", payload.SessionID,
					"phone_hash", sharedlogging.HashIdentifier(payload.CallerPhone),
				)
			} else {
				sharedlogging.Info("notification emergency escalation sms sent",
					"session_id", payload.SessionID,
					"phone_hash", sharedlogging.HashIdentifier(payload.CallerPhone),
				)
			}
		}

		if len(payload.AlertPhoneNumbers) > 0 {
			errs := c.svc.SendEmergencyEscalationAlerts(ctx, payload.AlertPhoneNumbers, payload.Reason)
			if len(errs) > 0 {
				for _, err := range errs {
					sharedlogging.Error("notification emergency escalation alert failed", err,
						"session_id", payload.SessionID,
					)
				}
			} else {
				sharedlogging.Info("notification emergency escalation alerts sent",
					"session_id", payload.SessionID,
					"alert_count", len(payload.AlertPhoneNumbers),
				)
			}
		}

		if payload.CallerPhone == "" && len(payload.AlertPhoneNumbers) == 0 {
			sharedlogging.Info("notification emergency escalation missing recipients", "session_id", payload.SessionID)
		}
		return nil
	})
}
