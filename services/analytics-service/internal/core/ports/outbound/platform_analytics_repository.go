package outbound

import (
	"context"

	models "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/domain/models"
)

// PlatformAnalyticsRepository is for Zorba admin queries — system-wide data.
type PlatformAnalyticsRepository interface {
	GetPlatformSummary(ctx context.Context) (*models.PlatformSummary, error)
	GetHospitalBreakdown(ctx context.Context, limit int, sortBy string) ([]models.HospitalStat, error)
}
