package outbound

import (
	"context"

	domain "github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/models"
)

// UserRepository defines persistence operations for users.
// The concrete Postgres/sqlc implementation lives in a secondary adapter in this service.
type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) (*domain.User, error)
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error)
	ResolveSessionActor(ctx context.Context, userID, userRole, sessionID string) (*domain.SessionActor, error)
	UpdateUserPassword(ctx context.Context, id, passwordHash string) error
	ListUsersByRole(ctx context.Context, role string, limit, offset int32) ([]*domain.User, error)
	DeleteUser(ctx context.Context, id string) error
}
