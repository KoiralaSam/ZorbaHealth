package outbound

import (
	"context"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
)

type PendingRegistrationRepository interface {
	Set(ctx context.Context, token string, data *models.PendingRegistration, ttl time.Duration) error
	Get(ctx context.Context, token string) (*models.PendingRegistration, error)
	Delete(ctx context.Context, token string) error
}
