-- name: GetHospitalStaffByID :one
SELECT id, hospital_id, email, name, role, active
FROM hospital_staff
WHERE id = $1 AND active = true;

-- name: ListSchedulableStaffByHospital :many
SELECT id, hospital_id, email, name, role
FROM hospital_staff
WHERE hospital_id = $1
  AND active = true
  AND role IN ('doctor', 'nurse')
ORDER BY name;

-- name: HasActivePatientHospitalConsent :one
SELECT EXISTS (
    SELECT 1
    FROM patient_hospital_consents
    WHERE patient_id = $1
      AND hospital_id = $2
      AND revoked_at IS NULL
) AS has_consent;

-- name: InsertScheduledMeeting :one
INSERT INTO scheduled_meetings (
    patient_id,
    staff_id,
    hospital_id,
    created_by_actor_type,
    created_by_actor_id,
    starts_at,
    duration_minutes,
    timezone,
    title,
    notes,
    join_url,
    status,
    correlation_id,
    voice_session_id,
    send_sms,
    channel
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
RETURNING *;

-- name: GetScheduledMeetingByID :one
SELECT *
FROM scheduled_meetings
WHERE id = $1;

-- name: ListScheduledMeetingsForPatient :many
SELECT *
FROM scheduled_meetings
WHERE patient_id = $1
  AND (sqlc.arg(include_cancelled)::boolean OR status IN ('pending', 'scheduled'))
ORDER BY starts_at DESC
LIMIT sqlc.arg(row_limit);

-- name: ListScheduledMeetingsForStaff :many
SELECT *
FROM scheduled_meetings
WHERE staff_id = $1
  AND (sqlc.arg(include_cancelled)::boolean OR status IN ('pending', 'scheduled'))
ORDER BY starts_at DESC
LIMIT sqlc.arg(row_limit);

-- name: ListScheduledMeetingsForHospital :many
SELECT *
FROM scheduled_meetings
WHERE hospital_id = $1
  AND (sqlc.arg(include_cancelled)::boolean OR status IN ('pending', 'scheduled'))
ORDER BY starts_at DESC
LIMIT sqlc.arg(row_limit);

-- name: CancelScheduledMeeting :one
UPDATE scheduled_meetings
SET status = 'cancelled'
WHERE id = $1 AND status IN ('pending', 'scheduled')
RETURNING *;

-- name: MarkScheduledMeeting :one
UPDATE scheduled_meetings
SET starts_at = $2,
    duration_minutes = $3,
    timezone = $4,
    title = $5,
    join_url = $6,
    status = 'scheduled'
WHERE id = $1 AND status = 'pending'
RETURNING *;
