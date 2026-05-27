package outbound

import (
	"context"

	domain "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
)

type PatientRepository interface {
	CreatePatient(ctx context.Context, patient *domain.Patient) (*domain.Patient, error)
	GetPatientByID(ctx context.Context, id string) (*domain.Patient, error)
	GetPatientByUserID(ctx context.Context, userID string) (*domain.Patient, error)
	GetPatientByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.Patient, error)
	GetPatientByEmail(ctx context.Context, email string) (*domain.Patient, error)
	GetPatientProfile(ctx context.Context, patientID string) (*domain.PatientProfile, error)
	ListPatientCallSummaries(ctx context.Context, patientID string, limit, offset int32) ([]domain.CallSummary, error)
	UpdatePatient(ctx context.Context, patient *domain.Patient) error
	DeletePatient(ctx context.Context, id string) error
}
