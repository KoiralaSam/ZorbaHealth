package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
)

type MeetingRepository struct {
	db *pgxpool.Pool
}

func NewMeetingRepository(db *pgxpool.Pool) outbound.MeetingRepository {
	return &MeetingRepository{db: db}
}

func (r *MeetingRepository) Insert(ctx context.Context, m *models.ScheduledMeeting) (*models.ScheduledMeeting, error) {
	row := r.db.QueryRow(ctx, `
INSERT INTO scheduled_meetings (
  patient_id, staff_id, hospital_id, created_by_actor_type, created_by_actor_id,
  starts_at, duration_minutes, timezone, title, notes,
  join_url, status, correlation_id, voice_session_id, send_sms, channel,
  livekit_room_name, livekit_room_sid, patient_token, staff_token
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
RETURNING id, created_at`,
		m.PatientID, m.StaffID, m.HospitalID, m.CreatedByActorType, m.CreatedByActorID,
		m.StartsAt, m.DurationMinutes, m.Timezone, m.Title, nullString(m.Notes),
		nullString(m.JoinURL), string(m.Status), m.CorrelationID, nullString(m.VoiceSessionID), m.SendSMS, string(m.Channel),
		nullString(m.LiveKitRoomName), nullString(m.LiveKitRoomSID), nullString(m.PatientToken), nullString(m.StaffToken),
	)
	var id uuid.UUID
	var createdAt time.Time
	if err := row.Scan(&id, &createdAt); err != nil {
		return nil, err
	}
	out := *m
	out.ID = id
	out.CreatedAt = createdAt
	return &out, nil
}

func (r *MeetingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ScheduledMeeting, error) {
	return r.scanOne(ctx, `SELECT id, patient_id, staff_id, hospital_id, created_by_actor_type, created_by_actor_id,
starts_at, duration_minutes, timezone, title, COALESCE(notes,''), COALESCE(join_url,''),
status, correlation_id, COALESCE(voice_session_id,''), send_sms, channel, created_at,
COALESCE(livekit_room_name,''), COALESCE(livekit_room_sid,''), COALESCE(patient_token,''), COALESCE(staff_token,'')
FROM scheduled_meetings WHERE id = $1`, id)
}

func (r *MeetingRepository) List(ctx context.Context, filter models.ListMeetingsFilter) ([]models.ScheduledMeeting, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	statusClause := "status IN ('pending', 'scheduled')"
	if filter.IncludeCancelled {
		statusClause = "1=1"
	}
	var (
		rows pgx.Rows
		err  error
	)
	switch {
	case filter.PatientID != nil:
		rows, err = r.db.Query(ctx, fmt.Sprintf(`
SELECT id, patient_id, staff_id, hospital_id, created_by_actor_type, created_by_actor_id,
starts_at, duration_minutes, timezone, title, COALESCE(notes,''), COALESCE(join_url,''),
status, correlation_id, COALESCE(voice_session_id,''), send_sms, channel, created_at,
COALESCE(livekit_room_name,''), COALESCE(livekit_room_sid,''), COALESCE(patient_token,''), COALESCE(staff_token,'')
FROM scheduled_meetings WHERE patient_id = $1 AND %s ORDER BY starts_at DESC LIMIT $2`, statusClause),
			*filter.PatientID, limit)
	case filter.StaffID != nil:
		rows, err = r.db.Query(ctx, fmt.Sprintf(`
SELECT id, patient_id, staff_id, hospital_id, created_by_actor_type, created_by_actor_id,
starts_at, duration_minutes, timezone, title, COALESCE(notes,''), COALESCE(join_url,''),
status, correlation_id, COALESCE(voice_session_id,''), send_sms, channel, created_at,
COALESCE(livekit_room_name,''), COALESCE(livekit_room_sid,''), COALESCE(patient_token,''), COALESCE(staff_token,'')
FROM scheduled_meetings WHERE staff_id = $1 AND %s ORDER BY starts_at DESC LIMIT $2`, statusClause),
			*filter.StaffID, limit)
	case filter.HospitalID != nil:
		rows, err = r.db.Query(ctx, fmt.Sprintf(`
SELECT id, patient_id, staff_id, hospital_id, created_by_actor_type, created_by_actor_id,
starts_at, duration_minutes, timezone, title, COALESCE(notes,''), COALESCE(join_url,''),
status, correlation_id, COALESCE(voice_session_id,''), send_sms, channel, created_at,
COALESCE(livekit_room_name,''), COALESCE(livekit_room_sid,''), COALESCE(patient_token,''), COALESCE(staff_token,'')
FROM scheduled_meetings WHERE hospital_id = $1 AND %s ORDER BY starts_at DESC LIMIT $2`, statusClause),
			*filter.HospitalID, limit)
	default:
		return nil, fmt.Errorf("list meetings filter requires patient_id, staff_id, or hospital_id")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRows(rows)
}

func (r *MeetingRepository) Cancel(ctx context.Context, id uuid.UUID) (*models.ScheduledMeeting, error) {
	return r.scanOne(ctx, `
UPDATE scheduled_meetings SET status = 'cancelled'
WHERE id = $1 AND status IN ('pending', 'scheduled')
RETURNING id, patient_id, staff_id, hospital_id, created_by_actor_type, created_by_actor_id,
starts_at, duration_minutes, timezone, title, COALESCE(notes,''), COALESCE(join_url,''),
status, correlation_id, COALESCE(voice_session_id,''), send_sms, channel, created_at,
COALESCE(livekit_room_name,''), COALESCE(livekit_room_sid,''), COALESCE(patient_token,''), COALESCE(staff_token,'')`, id)
}

func (r *MeetingRepository) MarkScheduled(ctx context.Context, m *models.ScheduledMeeting) (*models.ScheduledMeeting, error) {
	if m == nil {
		return nil, domainErrors.ErrMeetingNotFound
	}
	return r.scanOne(ctx, `
UPDATE scheduled_meetings
SET starts_at = $2,
    duration_minutes = $3,
    timezone = $4,
    title = $5,
    join_url = $6,
    livekit_room_name = $7,
    livekit_room_sid = $8,
    patient_token = $9,
    staff_token = $10,
    status = 'scheduled'
WHERE id = $1 AND status = 'pending'
RETURNING id, patient_id, staff_id, hospital_id, created_by_actor_type, created_by_actor_id,
starts_at, duration_minutes, timezone, title, COALESCE(notes,''), COALESCE(join_url,''),
status, correlation_id, COALESCE(voice_session_id,''), send_sms, channel, created_at,
COALESCE(livekit_room_name,''), COALESCE(livekit_room_sid,''), COALESCE(patient_token,''), COALESCE(staff_token,'')`,
		m.ID,
		m.StartsAt,
		m.DurationMinutes,
		m.Timezone,
		m.Title,
		nullString(m.JoinURL),
		nullString(m.LiveKitRoomName),
		nullString(m.LiveKitRoomSID),
		nullString(m.PatientToken),
		nullString(m.StaffToken),
	)
}

func (r *MeetingRepository) HasActiveConsent(ctx context.Context, patientID, hospitalID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM patient_hospital_consents
  WHERE patient_id = $1 AND hospital_id = $2 AND revoked_at IS NULL
)`, patientID, hospitalID).Scan(&ok)
	return ok, err
}

func (r *MeetingRepository) GetStaffByID(ctx context.Context, staffID uuid.UUID) (*models.StaffSummary, error) {
	var s models.StaffSummary
	var active bool
	err := r.db.QueryRow(ctx, `
SELECT id, hospital_id, name, role, email, active FROM hospital_staff WHERE id = $1`, staffID).
		Scan(&s.StaffID, &s.HospitalID, &s.Name, &s.Role, &s.Email, &active)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainErrors.ErrMeetingStaffNotFound
		}
		return nil, err
	}
	if !active {
		return nil, domainErrors.ErrMeetingStaffInactive
	}
	return &s, nil
}

func (r *MeetingRepository) ListSchedulableStaff(ctx context.Context, hospitalID uuid.UUID) ([]models.StaffSummary, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, hospital_id, name, role, email
FROM hospital_staff
WHERE hospital_id = $1 AND active = true AND role IN ('doctor', 'nurse')
ORDER BY name`, hospitalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.StaffSummary
	for rows.Next() {
		var s models.StaffSummary
		if err := rows.Scan(&s.StaffID, &s.HospitalID, &s.Name, &s.Role, &s.Email); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *MeetingRepository) scanOne(ctx context.Context, query string, args ...any) (*models.ScheduledMeeting, error) {
	row := r.db.QueryRow(ctx, query, args...)
	m, err := scanMeetingRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainErrors.ErrMeetingNotFound
		}
		return nil, err
	}
	return m, nil
}

func (r *MeetingRepository) scanRows(rows pgx.Rows) ([]models.ScheduledMeeting, error) {
	var out []models.ScheduledMeeting
	for rows.Next() {
		m, err := scanMeetingRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanMeetingRow(row scannable) (*models.ScheduledMeeting, error) {
	var m models.ScheduledMeeting
	var status, channel string
	err := row.Scan(
		&m.ID, &m.PatientID, &m.StaffID, &m.HospitalID,
		&m.CreatedByActorType, &m.CreatedByActorID,
		&m.StartsAt, &m.DurationMinutes, &m.Timezone, &m.Title, &m.Notes,
		&m.JoinURL,
		&status, &m.CorrelationID, &m.VoiceSessionID, &m.SendSMS, &channel, &m.CreatedAt,
		&m.LiveKitRoomName, &m.LiveKitRoomSID, &m.PatientToken, &m.StaffToken,
	)
	if err != nil {
		return nil, err
	}
	m.Status = models.MeetingStatus(status)
	m.Channel = models.MeetingChannel(channel)
	return &m, nil
}

func nullString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
