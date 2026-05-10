package outbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
)

type AuthRepository interface {
	Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResult, error)
	RegisterPatient(ctx context.Context, req *models.RegisterPatientRequest) (*models.RegisterResult, error)
}
