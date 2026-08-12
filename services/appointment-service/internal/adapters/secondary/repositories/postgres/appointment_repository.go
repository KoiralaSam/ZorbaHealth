package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/ports/outbound"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AppointmentRepository struct {
	db *pgxpool.Pool
}

func NewAppointmentRepository(db *pgxpool.Pool) outbound.AppointmentRepository {
	return &AppointmentRepository{db: db}
}

func (r *AppointmentRepository) Create(ctx context.Context, appt *models.Appointment) (*models.Appointment, error) {
	row := r.db.QueryRow(ctx, `
INSERT INTO appointments (
  id, patient_id, staff_id, hospital_id, time_range, duration_minutes, timezone,
  type, status, channel, title, notes, correlation_id, voice_session_id,
  booked_by_actor_type, booked_by_actor_id, join_url, livekit_room_name,
  livekit_room_sid, patient_token, staff_token
) VALUES (
  $1,$2,$3,$4, tstzrange($5,$6,'[)'), $7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22
)
RETURNING created_at, updated_at`,
		appt.ID, appt.PatientID, appt.StaffID, appt.HospitalID,
		appt.StartsAt, appt.EndsAt, appt.DurationMinutes, appt.Timezone,
		string(appt.Type), string(appt.Status), string(appt.Channel), appt.Title, nullStr(appt.Notes),
		appt.CorrelationID, nullStr(appt.VoiceSessionID), appt.BookedByActorType, appt.BookedByActorID,
		nullStr(appt.JoinURL), nullStr(appt.LiveKitRoomName), nullStr(appt.LiveKitRoomSID),
		nullStr(appt.PatientToken), nullStr(appt.StaffToken),
	)
	if err := row.Scan(&appt.CreatedAt, &appt.UpdatedAt); err != nil {
		return nil, mapPGError(err)
	}
	return appt, nil
}

func (r *AppointmentRepository) Update(ctx context.Context, appt *models.Appointment) (*models.Appointment, error) {
	row := r.db.QueryRow(ctx, `
UPDATE appointments SET
  time_range = tstzrange($2,$3,'[)'),
  duration_minutes = $4,
  timezone = $5,
  type = $6,
  status = $7,
  channel = $8,
  title = $9,
  notes = $10,
  join_url = $11,
  livekit_room_name = $12,
  livekit_room_sid = $13,
  patient_token = $14,
  staff_token = $15,
  updated_at = now()
WHERE id = $1
RETURNING updated_at`,
		appt.ID, appt.StartsAt, appt.EndsAt, appt.DurationMinutes, appt.Timezone,
		string(appt.Type), string(appt.Status), string(appt.Channel), appt.Title, nullStr(appt.Notes),
		nullStr(appt.JoinURL), nullStr(appt.LiveKitRoomName), nullStr(appt.LiveKitRoomSID),
		nullStr(appt.PatientToken), nullStr(appt.StaffToken),
	)
	if err := row.Scan(&appt.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, mapPGError(err)
	}
	return appt, nil
}

func (r *AppointmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Appointment, error) {
	return r.scanOne(ctx, `
SELECT id, patient_id, staff_id, hospital_id,
       lower(time_range), upper(time_range), duration_minutes, timezone,
       type, status, channel, title, COALESCE(notes,''), correlation_id,
       COALESCE(voice_session_id,''), booked_by_actor_type, booked_by_actor_id,
       COALESCE(join_url,''), COALESCE(livekit_room_name,''), COALESCE(livekit_room_sid,''),
       COALESCE(patient_token,''), COALESCE(staff_token,''), created_at, updated_at
FROM appointments WHERE id = $1`, id)
}

func (r *AppointmentRepository) List(ctx context.Context, filter models.ListAppointmentsFilter) ([]models.Appointment, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	statusClause := "status = 'booked'"
	if filter.IncludeCancelled {
		statusClause = "1=1"
	}
	conds := []string{statusClause}
	args := make([]any, 0, 4)
	argN := 1
	if filter.PatientID != nil {
		conds = append(conds, fmt.Sprintf("patient_id = $%d", argN))
		args = append(args, *filter.PatientID)
		argN++
	}
	if filter.StaffID != nil {
		conds = append(conds, fmt.Sprintf("staff_id = $%d", argN))
		args = append(args, *filter.StaffID)
		argN++
	}
	if filter.HospitalID != nil {
		conds = append(conds, fmt.Sprintf("hospital_id = $%d", argN))
		args = append(args, *filter.HospitalID)
		argN++
	}
	args = append(args, limit)
	q := fmt.Sprintf(`
SELECT id, patient_id, staff_id, hospital_id,
       lower(time_range), upper(time_range), duration_minutes, timezone,
       type, status, channel, title, COALESCE(notes,''), correlation_id,
       COALESCE(voice_session_id,''), booked_by_actor_type, booked_by_actor_id,
       COALESCE(join_url,''), COALESCE(livekit_room_name,''), COALESCE(livekit_room_sid,''),
       COALESCE(patient_token,''), COALESCE(staff_token,''), created_at, updated_at
FROM appointments
WHERE %s
ORDER BY lower(time_range) DESC
LIMIT $%d`, strings.Join(conds, " AND "), argN)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Appointment, 0)
	for rows.Next() {
		a, err := scanAppointment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *AppointmentRepository) ListBookedOverlapping(ctx context.Context, staffID uuid.UUID, from, to time.Time) ([]models.Appointment, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, patient_id, staff_id, hospital_id,
       lower(time_range), upper(time_range), duration_minutes, timezone,
       type, status, channel, title, COALESCE(notes,''), correlation_id,
       COALESCE(voice_session_id,''), booked_by_actor_type, booked_by_actor_id,
       COALESCE(join_url,''), COALESCE(livekit_room_name,''), COALESCE(livekit_room_sid,''),
       COALESCE(patient_token,''), COALESCE(staff_token,''), created_at, updated_at
FROM appointments
WHERE staff_id = $1
  AND status = 'booked'
  AND time_range && tstzrange($2,$3,'[)')`, staffID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Appointment, 0)
	for rows.Next() {
		a, err := scanAppointment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *AppointmentRepository) GetPatientContact(ctx context.Context, patientID uuid.UUID) (*models.PatientContact, error) {
	var c models.PatientContact
	err := r.db.QueryRow(ctx, `
SELECT id, COALESCE(email::text,''), COALESCE(phone_number,''), COALESCE(full_name,'')
FROM patients WHERE id = $1`, patientID).Scan(&c.ID, &c.Email, &c.PhoneNumber, &c.FullName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *AppointmentRepository) GetStaffContact(ctx context.Context, staffID uuid.UUID) (*models.StaffContact, error) {
	var c models.StaffContact
	err := r.db.QueryRow(ctx, `
SELECT id, hospital_id, COALESCE(email,''), COALESCE(name,''), COALESCE(phone_number,'')
FROM hospital_staff WHERE id = $1`, staffID).Scan(&c.ID, &c.HospitalID, &c.Email, &c.Name, &c.PhoneNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *AppointmentRepository) GetHospitalContact(ctx context.Context, hospitalID uuid.UUID) (*models.HospitalContact, error) {
	var c models.HospitalContact
	err := r.db.QueryRow(ctx, `
SELECT id,
       COALESCE(name,''),
       COALESCE(address,''),
       COALESCE(city,''),
       COALESCE(state,''),
       COALESCE(postal_code,''),
       COALESCE(phone,'')
FROM hospitals WHERE id = $1`, hospitalID).Scan(
		&c.ID, &c.Name, &c.Address, &c.City, &c.State, &c.PostalCode, &c.Phone,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *AppointmentRepository) HasActiveHospitalConsent(ctx context.Context, patientID, hospitalID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM patient_hospital_consents
  WHERE patient_id = $1 AND hospital_id = $2 AND revoked_at IS NULL
)`, patientID, hospitalID).Scan(&ok)
	return ok, err
}

func (r *AppointmentRepository) scanOne(ctx context.Context, q string, args ...any) (*models.Appointment, error) {
	row := r.db.QueryRow(ctx, q, args...)
	a, err := scanAppointment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return a, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanAppointment(row scannable) (*models.Appointment, error) {
	var a models.Appointment
	var typ, status, channel string
	if err := row.Scan(
		&a.ID, &a.PatientID, &a.StaffID, &a.HospitalID,
		&a.StartsAt, &a.EndsAt, &a.DurationMinutes, &a.Timezone,
		&typ, &status, &channel, &a.Title, &a.Notes, &a.CorrelationID,
		&a.VoiceSessionID, &a.BookedByActorType, &a.BookedByActorID,
		&a.JoinURL, &a.LiveKitRoomName, &a.LiveKitRoomSID,
		&a.PatientToken, &a.StaffToken, &a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return nil, err
	}
	a.Type = models.AppointmentType(typ)
	a.Status = models.AppointmentStatus(status)
	a.Channel = models.AppointmentChannel(channel)
	return &a, nil
}

func nullStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func mapPGError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23P01" { // exclusion_violation
			return domainerrors.ErrConflict
		}
	}
	return err
}
