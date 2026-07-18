package messaging

import (
	"encoding/json"
	"time"
)

type EventEnvelope struct {
	EventID         string          `json:"event_id"`
	EventType       string          `json:"event_type"`
	EventVersion    string          `json:"event_version"`
	Timestamp       time.Time       `json:"timestamp"`
	ProducerService string          `json:"producer_service"`
	CorrelationID   string          `json:"correlation_id,omitempty"`
	CausationID     string          `json:"causation_id,omitempty"`
	PatientID       string          `json:"patient_id,omitempty"`
	IdempotencyKey  string          `json:"idempotency_key,omitempty"`
	Payload         json.RawMessage `json:"payload"`
}

func NewEnvelope(eventType, producerService string, payload json.RawMessage) EventEnvelope {
	return EventEnvelope{
		EventType:       eventType,
		EventVersion:    "v1",
		Timestamp:       time.Now().UTC(),
		ProducerService: producerService,
		Payload:         payload,
	}
}
