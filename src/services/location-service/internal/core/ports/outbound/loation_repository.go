package outbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
)

// LocationRepository is the secondary port for persistence.
// Only implementation today: Redis. Could swap to another store without touching core.
type LocationRepository interface {
	Save(ctx context.Context, sessionID string, loc models.Location) error
	Get(ctx context.Context, sessionID string) (*models.Location, error)
	Delete(ctx context.Context, sessionID string) error
}
