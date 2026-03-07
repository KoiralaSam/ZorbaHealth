package mappers

import (
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
)

// ToProtoMapper maps domain models to patient gRPC proto types (e.g. LoginResponse when implemented).
type ToProtoMapper struct {
	Patient         *models.Patient
	RegisterRequest *models.RegisterPatientRequest
}

// NewToProtoMapper creates a mapper for the given patient and/or register request.
func NewToProtoMapper(patient *models.Patient, registerRequest *models.RegisterPatientRequest) *ToProtoMapper {
	return &ToProtoMapper{Patient: patient, RegisterRequest: registerRequest}
}
