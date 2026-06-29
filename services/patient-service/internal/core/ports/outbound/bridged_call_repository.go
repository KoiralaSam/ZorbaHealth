package outbound

import (
	"context"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
)

type BridgedCallRepository interface {
	Put(ctx context.Context, session *models.BridgedCallSession, ttl time.Duration) error
	Get(ctx context.Context, sessionID string) (*models.BridgedCallSession, error)
	List(ctx context.Context, hospitalID string, status models.BridgedCallStatus, limit int) ([]*models.BridgedCallSession, error)
}
