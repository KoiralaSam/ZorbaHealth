package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlc "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/adapters/secondary/repositories/postgres/sqlc"
	models "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/domain/models"
	outbound "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/ports/outbound"
)

var (
	_ outbound.HospitalAnalyticsRepository = (*HospitalAnalyticsRepository)(nil)
	_ outbound.PlatformAnalyticsRepository = (*PlatformAnalyticsRepository)(nil)
	_ outbound.PatientAnalyticsRepository  = (*PatientAnalyticsRepository)(nil)
)

type HospitalAnalyticsRepository struct {
	queries *sqlc.Queries
}

func NewHospitalAnalyticsRepository(db *pgxpool.Pool) *HospitalAnalyticsRepository {
	return &HospitalAnalyticsRepository{queries: sqlc.New(db)}
}

func (r *HospitalAnalyticsRepository) GetSummary(ctx context.Context, hospitalID string) (*models.HospitalSummary, error) {
	row, err := r.queries.GetHospitalSummary(ctx, hospitalID)
	if err != nil {
		return nil, err
	}

	return &models.HospitalSummary{
		HospitalID:             row.HospitalID,
		TotalConsentedPatients: int(row.TotalConsentedPatients),
		TotalCalls30d:          int(row.TotalCalls30d),
		EmergencyEvents30d:     int(row.EmergencyEvents30d),
		AvgCallDurationSeconds: row.AvgCallDurationSeconds,
		RecordsIndexed:         int(row.RecordsIndexed),
		ActivePatients7d:       int(row.ActivePatients7d),
	}, nil
}

func (r *HospitalAnalyticsRepository) GetCallVolume(
	ctx context.Context,
	hospitalID string,
	from, to time.Time,
	granularity string,
) ([]models.CallVolumePoint, error) {
	hospitalUUID, err := parseUUID(hospitalID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.GetHospitalCallVolume(ctx, sqlc.GetHospitalCallVolumeParams{
		Granularity: granularity,
		FromDate:    pgTimestamp(from),
		ToDate:      pgTimestamp(to),
		HospitalID:  hospitalUUID,
	})
	if err != nil {
		return nil, err
	}

	points := make([]models.CallVolumePoint, 0, len(rows))
	for _, row := range rows {
		date, err := interfaceToTime(row.Date)
		if err != nil {
			return nil, err
		}

		points = append(points, models.CallVolumePoint{
			Date:           date,
			TotalCalls:     int(row.TotalCalls),
			CompletedCalls: int(row.CompletedCalls),
			EmergencyCalls: int(row.EmergencyCalls),
			AvgDurationSec: row.AvgDurationSec,
		})
	}

	return points, nil
}

func (r *HospitalAnalyticsRepository) GetTopConditions(ctx context.Context, hospitalID string, limit int) ([]models.ConditionCount, error) {
	hospitalUUID, err := parseUUID(hospitalID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.GetHospitalTopConditions(ctx, sqlc.GetHospitalTopConditionsParams{
		HospitalID:  hospitalUUID,
		ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	conditions := make([]models.ConditionCount, 0, len(rows))
	for _, row := range rows {
		conditionName, err := interfaceToString(row.ConditionName)
		if err != nil {
			return nil, err
		}

		conditions = append(conditions, models.ConditionCount{
			ConditionName: conditionName,
			PatientCount:  int(row.PatientCount),
			Percentage:    row.Percentage,
		})
	}

	return conditions, nil
}

func (r *HospitalAnalyticsRepository) GetToolUsage(ctx context.Context, hospitalID string, from time.Time) ([]models.ToolUsageStat, error) {
	rows, err := r.queries.GetHospitalToolUsage(ctx, sqlc.GetHospitalToolUsageParams{
		HospitalID: hospitalID,
		FromTime:   pgTimestamptz(from),
	})
	if err != nil {
		return nil, err
	}

	stats := make([]models.ToolUsageStat, 0, len(rows))
	for _, row := range rows {
		stats = append(stats, models.ToolUsageStat{
			Tool:         row.Tool,
			SuccessCount: int(row.SuccessCount),
			ErrorCount:   int(row.ErrorCount),
			DeniedCount:  int(row.DeniedCount),
			SuccessRate:  row.SuccessRate,
		})
	}

	return stats, nil
}

func (r *HospitalAnalyticsRepository) GetRecentActivity(ctx context.Context, hospitalID string, limit int) ([]models.ActivityEvent, error) {
	rows, err := r.queries.GetHospitalRecentActivity(ctx, sqlc.GetHospitalRecentActivityParams{
		HospitalID:  hospitalID,
		ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	events := make([]models.ActivityEvent, 0, len(rows))
	for _, row := range rows {
		timestamp, err := pgTimestamptzToTime(row.Timestamp)
		if err != nil {
			return nil, err
		}

		events = append(events, models.ActivityEvent{
			Timestamp: timestamp,
			Tool:      row.Tool,
			ActorType: row.ActorType,
			Outcome:   row.Outcome,
			SessionID: row.SessionID,
		})
	}

	return events, nil
}

type PlatformAnalyticsRepository struct {
	queries *sqlc.Queries
}

func NewPlatformAnalyticsRepository(db *pgxpool.Pool) *PlatformAnalyticsRepository {
	return &PlatformAnalyticsRepository{queries: sqlc.New(db)}
}

func (r *PlatformAnalyticsRepository) GetPlatformSummary(ctx context.Context) (*models.PlatformSummary, error) {
	row, err := r.queries.GetPlatformSummary(ctx)
	if err != nil {
		return nil, err
	}

	return &models.PlatformSummary{
		TotalHospitals:      int(row.TotalHospitals),
		TotalPatients:       int(row.TotalPatients),
		TotalCalls30d:       int(row.TotalCalls30d),
		TotalEmergencies30d: int(row.TotalEmergencies30d),
		AvgCallDurationSec:  row.AvgCallDurationSec,
		ActiveHospitals7d:   int(row.ActiveHospitals7d),
	}, nil
}

func (r *PlatformAnalyticsRepository) GetHospitalBreakdown(ctx context.Context, limit int, sortBy string) ([]models.HospitalStat, error) {
	rows, err := r.queries.GetPlatformHospitalBreakdown(ctx, sqlc.GetPlatformHospitalBreakdownParams{
		SortBy:      sortBy,
		ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	stats := make([]models.HospitalStat, 0, len(rows))
	for _, row := range rows {
		stats = append(stats, models.HospitalStat{
			HospitalID:     row.HospitalID,
			HospitalName:   row.HospitalName,
			PatientCount:   int(row.PatientCount),
			CallCount:      int(row.CallCount),
			EmergencyCount: int(row.EmergencyCount),
		})
	}

	return stats, nil
}

type PatientAnalyticsRepository struct {
	queries *sqlc.Queries
}

func NewPatientAnalyticsRepository(db *pgxpool.Pool) *PatientAnalyticsRepository {
	return &PatientAnalyticsRepository{queries: sqlc.New(db)}
}

func (r *PatientAnalyticsRepository) GetPatientCallHistory(ctx context.Context, patientID string, limit int) (*models.PatientCallHistory, error) {
	patientUUID, err := parseUUID(patientID)
	if err != nil {
		return nil, err
	}

	totalCalls, err := r.queries.CountPatientCalls(ctx, patientUUID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.GetPatientCallHistoryRows(ctx, sqlc.GetPatientCallHistoryRowsParams{
		PatientID:   patientUUID,
		ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	calls := make([]models.PatientCall, 0, len(rows))
	for _, row := range rows {
		startedAt, err := pgTimestampToTime(row.StartedAt)
		if err != nil {
			return nil, err
		}

		duration, err := interfaceToString(row.Duration)
		if err != nil {
			return nil, err
		}

		calls = append(calls, models.PatientCall{
			StartedAt:    startedAt,
			Duration:     duration,
			Summary:      row.Summary,
			HadEmergency: row.HadEmergency,
		})
	}

	return &models.PatientCallHistory{
		TotalCalls: int(totalCalls),
		Calls:      calls,
	}, nil
}

func parseUUID(raw string) (pgtype.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID %q: %w", raw, err)
	}

	var bytes [16]byte
	copy(bytes[:], id[:])

	return pgtype.UUID{
		Bytes: bytes,
		Valid: true,
	}, nil
}

func pgTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{
		Time:  t,
		Valid: true,
	}
}

func pgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  t,
		Valid: true,
	}
}

func pgTimestampToTime(ts pgtype.Timestamp) (time.Time, error) {
	if !ts.Valid {
		return time.Time{}, fmt.Errorf("timestamp is null")
	}
	return ts.Time, nil
}

func pgTimestamptzToTime(ts pgtype.Timestamptz) (time.Time, error) {
	if !ts.Valid {
		return time.Time{}, fmt.Errorf("timestamp with time zone is null")
	}
	return ts.Time, nil
}

func interfaceToTime(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case pgtype.Timestamp:
		return pgTimestampToTime(v)
	case pgtype.Timestamptz:
		return pgTimestamptzToTime(v)
	default:
		return time.Time{}, fmt.Errorf("unsupported time type %T", value)
	}
}

func interfaceToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return "", fmt.Errorf("unsupported string type %T", value)
	}
}
