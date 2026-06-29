package outbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
)

type AuthRepository interface {
	Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResult, error)
	ValidateUserCredentials(ctx context.Context, req *models.LoginRequest) (userID, role string, err error)
	RegisterPatient(ctx context.Context, req *models.RegisterPatientRequest) (*models.RegisterResult, error)
	CreatePatientSession(ctx context.Context, userID string, scopes []string) (*models.LoginResult, error)
}
