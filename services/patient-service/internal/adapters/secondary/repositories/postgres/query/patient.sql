-- name: CreatePatient :one
INSERT INTO patients (
  user_id,
  phone_number,
  email,
  full_name,
  date_of_birth,
  medical_notes,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetPatientByID :one
SELECT * FROM patients
WHERE id = $1 LIMIT 1;

-- name: GetPatientByPhoneNumber :one
SELECT * FROM patients
WHERE phone_number = $1 LIMIT 1;

-- name: GetPatientByEmail :one
SELECT * FROM patients
WHERE email = $1 LIMIT 1;

-- name: GetPatientByUserID :one
SELECT * FROM patients
WHERE user_id = $1 LIMIT 1;

-- name: ListPatients :many
SELECT * FROM patients
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: SearchPatientsByName :many
SELECT * FROM patients
WHERE full_name ILIKE '%' || $1 || '%'
ORDER BY full_name
LIMIT $2 OFFSET $3;

-- name: UpdatePatient :one
UPDATE patients
SET
  email = COALESCE(sqlc.narg('email'), email),
  full_name = COALESCE(sqlc.narg('full_name'), full_name),
  date_of_birth = COALESCE(sqlc.narg('date_of_birth'), date_of_birth),
  medical_notes = COALESCE(sqlc.narg('medical_notes'), medical_notes),
  updated_at = $2
WHERE id = $1
RETURNING *;

-- name: UpdatePatientMedicalNotes :one
UPDATE patients
SET
  medical_notes = $2,
  updated_at = $3
WHERE id = $1
RETURNING *;

-- name: DeletePatient :exec
DELETE FROM patients
WHERE id = $1;

-- name: CountPatients :one
SELECT COUNT(*) FROM patients;

-- name: GetPatientsWithRecentCalls :many
SELECT DISTINCT p.*
FROM patients p
INNER JOIN calls c ON p.id = c.patient_id
WHERE c.started_at >= $1
ORDER BY p.full_name;
