package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
)

type PatientPublisher struct {
	rabbitmq *messaging.RabbitMQ
}

func NewPatientPublisher(rmq *messaging.RabbitMQ) *PatientPublisher {
	return &PatientPublisher{rabbitmq: rmq}
}

func (p *PatientPublisher) PublishPatientRegistered(ctx context.Context, patient *models.Patient) error {
	payload := events.PatientEventData{
		Patient: &events.PatientRegisteredData{
			Message:   "Patient registered successfully",
			PatientID: patient.ID.String(),
			UserID:    patient.UserID.String(),
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.rabbitmq.PublishMessage(ctx, messaging.PatientExchange, contracts.PatientEventRegistered, contracts.AmqpMessage{
		OwnerID: patient.ID.String(),
		Data:    jsonData,
	})
}

func (p *PatientPublisher) PublishPatientNotRegistered(ctx context.Context, patientRegisterRequest *models.Patient) error {
	payload := events.PatientEventData{
		RegisterRequest: &events.PendingRegistrationData{
			Email:       patientRegisterRequest.Email,
			PhoneNumber: patientRegisterRequest.PhoneNumber,
			FullName:    patientRegisterRequest.FullName,
			DateOfBirth: patientRegisterRequest.DateOfBirth.Format(time.RFC3339),
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.rabbitmq.PublishMessage(ctx, messaging.PatientExchange, contracts.PatientEventNotRegistered, contracts.AmqpMessage{
		OwnerID: patientRegisterRequest.PhoneNumber,
		Data:    jsonData,
	})
}

func (p *PatientPublisher) PublishPatientChached(ctx context.Context, patientRegisterRequest *models.RegisterPatientRequest, token string, otp string) error {
	payload := events.PatientEventData{
		RegisterRequest: &events.PendingRegistrationData{
			Email:       patientRegisterRequest.Email,
			PhoneNumber: patientRegisterRequest.PhoneNumber,
			FullName:    patientRegisterRequest.FullName,
			DateOfBirth: patientRegisterRequest.DateOfBirth.Format(time.RFC3339),
			Otp:         otp,
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.rabbitmq.PublishMessage(ctx, messaging.PatientExchange, contracts.PatientEventChached, contracts.AmqpMessage{
		OwnerID: token,
		Data:    jsonData,
	})
}
