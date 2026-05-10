-- name: CreateCall :one
INSERT INTO calls (
  patient_id,
  livekit_room_id,
  status,
  started_at
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetCallByID :one
SELECT * FROM calls
WHERE id = $1 LIMIT 1;

-- name: GetCallByLivekitRoomID :one
SELECT * FROM calls
WHERE livekit_room_id = $1 LIMIT 1;

-- name: ListCallsByPatientID :many
SELECT * FROM calls
WHERE patient_id = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3;

-- name: ListCallsByStatus :many
SELECT * FROM calls
WHERE status = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3;

-- name: GetRecentCalls :many
SELECT * FROM calls
ORDER BY started_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateCallStatus :one
UPDATE calls
SET status = $2
WHERE id = $1
RETURNING *;

-- name: EndCall :one
UPDATE calls
SET
  status = 'ended',
  ended_at = $2,
  recording_s3_url = COALESCE(sqlc.narg('recording_s3_url'), recording_s3_url),
  summary = COALESCE(sqlc.narg('summary'), summary)
WHERE id = $1
RETURNING *;

-- name: UpdateCallSummary :one
UPDATE calls
SET summary = $2
WHERE id = $1
RETURNING *;

-- name: UpdateCallRecordingURL :one
UPDATE calls
SET recording_s3_url = $2
WHERE id = $1
RETURNING *;

-- name: GetCallsInDateRange :many
SELECT * FROM calls
WHERE started_at BETWEEN $1 AND $2
ORDER BY started_at DESC;

-- name: CountCallsByPatientID :one
SELECT COUNT(*) FROM calls
WHERE patient_id = $1;

-- name: GetCallWithPatient :one
SELECT 
  c.*,
  p.phone_number,
  p.full_name as patient_name,
  p.email as patient_email
FROM calls c
INNER JOIN patients p ON c.patient_id = p.id
WHERE c.id = $1
LIMIT 1;

-- name: DeleteCall :exec
DELETE FROM calls
WHERE id = $1;
