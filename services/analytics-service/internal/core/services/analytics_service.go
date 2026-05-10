package services

import (
	"context"
	"time"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/domain/errors"
	models "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/domain/models"
	outbound "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/ports/outbound"
)

type AnalyticsService struct {
	hospitalRepo outbound.HospitalAnalyticsRepository
	platformRepo outbound.PlatformAnalyticsRepository
	patientRepo  outbound.PatientAnalyticsRepository
}

func NewAnalyticsService(
	hospitalRepo outbound.HospitalAnalyticsRepository,
	platformRepo outbound.PlatformAnalyticsRepository,
	patientRepo outbound.PatientAnalyticsRepository,
) *AnalyticsService {
	return &AnalyticsService{
		hospitalRepo: hospitalRepo,
		platformRepo: platformRepo,
		patientRepo:  patientRepo,
	}
}

func (s *AnalyticsService) GetHospitalSummary(ctx context.Context, hospitalID string) (*models.HospitalSummary, error) {
	if hospitalID == "" {
		return nil, domainerrors.ErrHospitalNotFound
	}
	return s.hospitalRepo.GetSummary(ctx, hospitalID)
}

func (s *AnalyticsService) GetCallVolume(ctx context.Context, hospitalID, period, granularity string) ([]models.CallVolumePoint, error) {
	from, err := parsePeriod(period)
	if err != nil {
		return nil, err
	}

	return s.hospitalRepo.GetCallVolume(ctx, hospitalID, from, time.Now(), normalizeGranularity(granularity))
}

func (s *AnalyticsService) GetTopConditions(ctx context.Context, hospitalID string, limit int) ([]models.ConditionCount, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.hospitalRepo.GetTopConditions(ctx, hospitalID, limit)
}

func (s *AnalyticsService) GetToolUsage(ctx context.Context, hospitalID, period string) ([]models.ToolUsageStat, error) {
	from, err := parsePeriod(period)
	if err != nil {
		return nil, err
	}
	return s.hospitalRepo.GetToolUsage(ctx, hospitalID, from)
}

func (s *AnalyticsService) GetRecentActivity(ctx context.Context, hospitalID string, limit int) ([]models.ActivityEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.hospitalRepo.GetRecentActivity(ctx, hospitalID, limit)
}

func (s *AnalyticsService) GetPlatformSummary(ctx context.Context) (*models.PlatformSummary, error) {
	return s.platformRepo.GetPlatformSummary(ctx)
}

func (s *AnalyticsService) GetHospitalBreakdown(ctx context.Context, limit int, sortBy string) ([]models.HospitalStat, error) {
	if limit <= 0 {
		limit = 20
	}
	validSort := map[string]bool{"calls": true, "patients": true, "emergencies": true}
	if !validSort[sortBy] {
		sortBy = "calls"
	}
	return s.platformRepo.GetHospitalBreakdown(ctx, limit, sortBy)
}

func (s *AnalyticsService) GetPatientCallHistory(ctx context.Context, patientID string, limit int) (*models.PatientCallHistory, error) {
	if patientID == "" {
		return nil, domainerrors.ErrPatientNotFound
	}
	if limit <= 0 {
		limit = 20
	}
	return s.patientRepo.GetPatientCallHistory(ctx, patientID, limit)
}

// parsePeriod converts "7d", "30d", "90d" into a time.Time boundary.
func parsePeriod(period string) (time.Time, error) {
	switch period {
	case "7d":
		return time.Now().AddDate(0, 0, -7), nil
	case "30d", "":
		return time.Now().AddDate(0, 0, -30), nil
	case "90d":
		return time.Now().AddDate(0, 0, -90), nil
	default:
		return time.Time{}, domainerrors.ErrInvalidPeriod
	}
}

func normalizeGranularity(granularity string) string {
	switch granularity {
	case "day", "week", "month":
		return granularity
	default:
		return "day"
	}
}
