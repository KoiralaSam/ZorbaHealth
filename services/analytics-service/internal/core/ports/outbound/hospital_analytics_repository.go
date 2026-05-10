package outbound

import (
	"context"
	"time"

	models "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/domain/models"
)

// HospitalAnalyticsRepository is the secondary port for hospital-scoped queries.
// Implementation reads from materialised views — never from live tables directly.
type HospitalAnalyticsRepository interface {
	GetSummary(ctx context.Context, hospitalID string) (*models.HospitalSummary, error)
	GetCallVolume(ctx context.Context, hospitalID string, from, to time.Time, granularity string) ([]models.CallVolumePoint, error)
	GetTopConditions(ctx context.Context, hospitalID string, limit int) ([]models.ConditionCount, error)
	GetToolUsage(ctx context.Context, hospitalID string, from time.Time) ([]models.ToolUsageStat, error)
	GetRecentActivity(ctx context.Context, hospitalID string, limit int) ([]models.ActivityEvent, error)
}
