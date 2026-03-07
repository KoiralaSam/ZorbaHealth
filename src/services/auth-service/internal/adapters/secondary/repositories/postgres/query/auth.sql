-- name: CreateAuth :one
INSERT INTO auths (user_id)
VALUES ($1)
RETURNING *;

-- name: FetchAuth :one
SELECT * FROM auths
WHERE id = $1 and user_id = $2 LIMIT 1;

-- name: DeleteAuth :exec
DELETE FROM auths
WHERE id = $1 and user_id = $2;

-- name: GetAuthByUserIDAndAuthUUID :one
SELECT * FROM auths
WHERE user_id = $1 AND auth_uuid = $2 LIMIT 1;

-- name: DeleteAuthByUserIDAndAuthUUID :exec
DELETE FROM auths
WHERE user_id = $1 AND auth_uuid = $2;
