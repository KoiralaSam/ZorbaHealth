package inbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/models"
)

// AuthService is the inbound port implemented by the core auth service.
// Primary adapters (gRPC/HTTP/consumers) should depend on this interface.
type AuthService interface {
	RegisterUser(ctx context.Context, email, phoneNumber, password, role string) (*models.User, error)
	Login(ctx context.Context, email, phoneNumber, password string) (token string, userID, role string, err error)
	CreateSession(ctx context.Context, userID string, authUUID string) (string, *models.Auth, error)
	Logout(ctx context.Context, accessToken string) (string, error)
	VerifyToken(ctx context.Context, accessToken string) (userID, authUUID, role string, err error)
	DeleteUser(ctx context.Context, id string) error
}

