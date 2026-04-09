package outbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
)

// PatientPublisher publishes patient lifecycle events (registration cached, registered, etc).
type PatientPublisher interface {
	PublishPatientRegistered(ctx context.Context, patient *models.Patient) error
	PublishPatientChached(ctx context.Context, patientRegisterRequest *models.RegisterPatientRequest, token string, otp string) error
	PublishPhoneVerificationCode(ctx context.Context, phone, fullName, otp string) error
}
