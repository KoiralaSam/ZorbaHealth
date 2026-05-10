-- name: CreateUser :one
INSERT INTO users (
  email,
  phone_number,
  password_hash,
  role
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByPhoneNumber :one
SELECT * FROM users
WHERE phone_number = $1 LIMIT 1;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = $2
WHERE id = $1
RETURNING *;

-- name: ListUsersByRole :many
SELECT * FROM users
WHERE role = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: GetPatientByUserID :one
SELECT id, user_id
FROM patients
WHERE user_id = $1
LIMIT 1;

-- name: GetHospitalStaffByUserID :one
SELECT id, hospital_id, user_id, role
FROM hospital_staff
WHERE user_id = $1
LIMIT 1;

-- name: GetAdminByUserID :one
SELECT id, user_id
FROM admins
WHERE user_id = $1
LIMIT 1;
