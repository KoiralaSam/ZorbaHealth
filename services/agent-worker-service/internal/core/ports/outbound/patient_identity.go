package outbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/domain/models"
)

type PatientIdentityClient interface {
	LookupByPhone(ctx context.Context, phone string) (*models.PatientCandidate, error)
	StartExistingPhoneVerification(ctx context.Context, phone string) error
	VerifyExistingPhoneOTP(ctx context.Context, phone, otp string) (*models.IdentifiedPatient, error)
	StartRegistration(ctx context.Context, req models.RegistrationRequest) (string, error)
	VerifyRegistrationOTPAndCreatePatient(ctx context.Context, phone, otp, registrationToken string) (*models.IdentifiedPatient, error)
}
