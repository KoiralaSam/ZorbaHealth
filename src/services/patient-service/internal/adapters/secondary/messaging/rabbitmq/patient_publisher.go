package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	outbound "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
)

type PatientPublisher struct {
	rabbitmq *messaging.RabbitMQ
}

func NewPatientPublisher(rmq *messaging.RabbitMQ) outbound.PatientPublisher {
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
	return p.rabbitmq.PublishMessage(ctx, events.PatientExchange, contracts.PatientEventRegistered, contracts.AmqpMessage{
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
	return p.rabbitmq.PublishMessage(ctx, events.PatientExchange, contracts.PatientEventNotRegistered, contracts.AmqpMessage{
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
	return p.rabbitmq.PublishMessage(ctx, events.PatientExchange, contracts.PatientEventChached, contracts.AmqpMessage{
		OwnerID: token,
		Data:    jsonData,
	})
}

func (p *PatientPublisher) PublishPhoneVerificationCode(ctx context.Context, phone, fullName, otp string) error {
	payload := events.PatientEventData{
		PhoneVerification: &events.PhoneVerificationData{
			PhoneNumber: phone,
			FullName:    fullName,
			Otp:         otp,
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.rabbitmq.PublishMessage(ctx, events.PatientExchange, contracts.PatientEventVerificationCodeRequested, contracts.AmqpMessage{
		OwnerID: phone,
		Data:    jsonData,
	})
}
