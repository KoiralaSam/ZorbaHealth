package inbound

import (
	"context"

	models "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/domain/models"
)

type AnalyticsService interface {
	// Hospital admin
	GetHospitalSummary(ctx context.Context, hospitalID string) (*models.HospitalSummary, error)
	GetCallVolume(ctx context.Context, hospitalID, period, granularity string) ([]models.CallVolumePoint, error)
	GetTopConditions(ctx context.Context, hospitalID string, limit int) ([]models.ConditionCount, error)
	GetToolUsage(ctx context.Context, hospitalID, period string) ([]models.ToolUsageStat, error)
	GetRecentActivity(ctx context.Context, hospitalID string, limit int) ([]models.ActivityEvent, error)

	// Platform admin
	GetPlatformSummary(ctx context.Context) (*models.PlatformSummary, error)
	GetHospitalBreakdown(ctx context.Context, limit int, sortBy string) ([]models.HospitalStat, error)

	// Patient
	GetPatientCallHistory(ctx context.Context, patientID string, limit int) (*models.PatientCallHistory, error)
}
