package outbound

import (
	"context"
	"time"
)

type RefreshTokenRecord struct {
	ID         int64
	AuthUUID   string
	UserID     string
	ActorType  string
	Generation int
	UsedAt     *time.Time
	RevokedAt  *time.Time
	ExpiresAt  time.Time
}

// RefreshRepository handles rotating refresh token persistence.
type RefreshRepository interface {
	InsertRefresh(ctx context.Context, authUUID, userID, actorType string, tokenHash []byte, generation int, expiresAt time.Time) error
	GetAuthUUIDByTokenHash(ctx context.Context, tokenHash []byte) (authUUID string, err error)
	RotateRefresh(ctx context.Context, presentedHash []byte, newHash []byte, newExpiresAt time.Time) (newGeneration int, authUUID, userID, actorType string, reuse bool, err error)
	RevokeFamily(ctx context.Context, authUUID, reason string) error
}
