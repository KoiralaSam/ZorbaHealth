package inbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
)

// PatientService is the inbound port implemented by the core service.
// Primary adapters (gRPC/HTTP) should depend on this interface.
type PatientService interface {
	StartRegistrationWithVerification(ctx context.Context, req *models.RegisterPatientRequest) (verificationToken string, otp string, err error)
	StartExistingPhoneVerification(ctx context.Context, phone string) error
	VerifyEmailAndCreatePatient(ctx context.Context, token string) (*models.Patient, error)
	VerifyPhoneOTP(ctx context.Context, phone string, code string) error
	VerifyExistingPhoneOTP(ctx context.Context, phone string, code string) (*models.Patient, error)
	CompletePhoneRegistration(ctx context.Context, token string) (*models.Patient, error)

	LoginPatient(ctx context.Context, patient *models.Patient) (*models.Patient, error)
	GetPatientByID(ctx context.Context, id string) (*models.Patient, error)
	GetPatientByPhoneNumber(ctx context.Context, phoneNumber string) (*models.Patient, error)
	GetPatientByEmail(ctx context.Context, email string) (*models.Patient, error)
	UpdatePatient(ctx context.Context, patient *models.Patient) error
	DeletePatient(ctx context.Context, id string) error
}
