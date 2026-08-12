package postgres

import (
	"context"
	"errors"
	"time"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/ports/outbound"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AvailabilityRepository struct {
	db *pgxpool.Pool
}

func NewAvailabilityRepository(db *pgxpool.Pool) outbound.AvailabilityRepository {
	return &AvailabilityRepository{db: db}
}

func (r *AvailabilityRepository) ReplaceRules(ctx context.Context, staffID, hospitalID uuid.UUID, rules []models.AvailabilityRule) ([]models.AvailabilityRule, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM staff_availability_rules WHERE staff_id = $1 AND hospital_id = $2`, staffID, hospitalID); err != nil {
		return nil, err
	}

	out := make([]models.AvailabilityRule, 0, len(rules))
	for _, rule := range rules {
		var id uuid.UUID
		var createdAt, updatedAt time.Time
		var effectiveUntil *time.Time
		err := tx.QueryRow(ctx, `
INSERT INTO staff_availability_rules (
  staff_id, hospital_id, weekday, start_time_local, end_time_local,
  slot_duration_minutes, timezone, effective_from, effective_until
) VALUES ($1,$2,$3,$4::time,$5::time,$6,$7,$8::date,$9)
RETURNING id, created_at, updated_at, effective_until`,
			staffID, hospitalID, rule.Weekday, rule.StartTimeLocal, rule.EndTimeLocal,
			rule.SlotDurationMinutes, rule.Timezone, rule.EffectiveFrom,
			nullableDate(rule.EffectiveUntil),
		).Scan(&id, &createdAt, &updatedAt, &effectiveUntil)
		if err != nil {
			return nil, err
		}
		rule.ID = id
		rule.StaffID = staffID
		rule.HospitalID = hospitalID
		rule.CreatedAt = createdAt
		rule.UpdatedAt = updatedAt
		rule.EffectiveUntil = effectiveUntil
		out = append(out, rule)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *AvailabilityRepository) ListRules(ctx context.Context, staffID, hospitalID uuid.UUID) ([]models.AvailabilityRule, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, staff_id, hospital_id, weekday,
       to_char(start_time_local, 'HH24:MI'), to_char(end_time_local, 'HH24:MI'),
       slot_duration_minutes, timezone, effective_from, effective_until, created_at, updated_at
FROM staff_availability_rules
WHERE staff_id = $1 AND hospital_id = $2
ORDER BY weekday, start_time_local`, staffID, hospitalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AvailabilityRule, 0)
	for rows.Next() {
		var rule models.AvailabilityRule
		var effectiveUntil *time.Time
		if err := rows.Scan(
			&rule.ID, &rule.StaffID, &rule.HospitalID, &rule.Weekday,
			&rule.StartTimeLocal, &rule.EndTimeLocal,
			&rule.SlotDurationMinutes, &rule.Timezone, &rule.EffectiveFrom, &effectiveUntil,
			&rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rule.EffectiveUntil = effectiveUntil
		out = append(out, rule)
	}
	return out, rows.Err()
}

func (r *AvailabilityRepository) AddException(ctx context.Context, ex models.AvailabilityException) (*models.AvailabilityException, error) {
	err := r.db.QueryRow(ctx, `
INSERT INTO staff_availability_exceptions (staff_id, hospital_id, time_range, reason, is_available)
VALUES ($1,$2,tstzrange($3,$4,'[)'),$5,$6)
RETURNING id, created_at`,
		ex.StaffID, ex.HospitalID, ex.StartsAt, ex.EndsAt, ex.Reason, ex.IsAvailable,
	).Scan(&ex.ID, &ex.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &ex, nil
}

func (r *AvailabilityRepository) RemoveException(ctx context.Context, exceptionID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM staff_availability_exceptions WHERE id = $1`, exceptionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *AvailabilityRepository) GetException(ctx context.Context, exceptionID uuid.UUID) (*models.AvailabilityException, error) {
	var ex models.AvailabilityException
	err := r.db.QueryRow(ctx, `
SELECT id, staff_id, hospital_id, lower(time_range), upper(time_range), COALESCE(reason,''), is_available, created_at
FROM staff_availability_exceptions WHERE id = $1`, exceptionID).Scan(
		&ex.ID, &ex.StaffID, &ex.HospitalID, &ex.StartsAt, &ex.EndsAt, &ex.Reason, &ex.IsAvailable, &ex.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &ex, nil
}

func (r *AvailabilityRepository) ListExceptions(ctx context.Context, staffID, hospitalID uuid.UUID, from, to time.Time) ([]models.AvailabilityException, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, staff_id, hospital_id, lower(time_range), upper(time_range), COALESCE(reason,''), is_available, created_at
FROM staff_availability_exceptions
WHERE staff_id = $1 AND hospital_id = $2
  AND time_range && tstzrange($3,$4,'[)')
ORDER BY lower(time_range)`, staffID, hospitalID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AvailabilityException, 0)
	for rows.Next() {
		var ex models.AvailabilityException
		if err := rows.Scan(&ex.ID, &ex.StaffID, &ex.HospitalID, &ex.StartsAt, &ex.EndsAt, &ex.Reason, &ex.IsAvailable, &ex.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ex)
	}
	return out, rows.Err()
}

func nullableDate(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.Format("2006-01-02")
}
