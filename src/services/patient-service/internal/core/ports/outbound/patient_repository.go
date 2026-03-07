package outbound

import (
	"context"

	domain "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
)

type PatientRepository interface {
	CreatePatient(ctx context.Context, patient *domain.Patient) (*domain.Patient, error)
	GetPatientByID(ctx context.Context, id string) (*domain.Patient, error)
	GetPatientByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.Patient, error)
	GetPatientByEmail(ctx context.Context, email string) (*domain.Patient, error)
	UpdatePatient(ctx context.Context, patient *domain.Patient) error
	DeletePatient(ctx context.Context, id string) error
}
