package outbound

import (
	"context"

	domain "github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/models"
)

// AuthRepository defines persistence operations for auth sessions.
// The concrete Postgres/sqlc implementation will live in a secondary adapter in this service.
type AuthRepository interface {
	CreateAuth(ctx context.Context, userID, authUUID string) (*domain.Auth, error)
	GetAuthByUserIDAndAuthUUID(ctx context.Context, userID, authUUID string) (*domain.Auth, error)
	DeleteAuth(ctx context.Context, userID, authUUID string) error
}

