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

-- name: GetAuthByAuthUUID :one
SELECT id, user_id, auth_uuid, revoked_at, revoked_reason, last_active_at, client_kind
FROM auths
WHERE auth_uuid = $1
LIMIT 1;

-- name: RevokeAuthByAuthUUID :exec
UPDATE auths
SET revoked_at = now(),
    revoked_reason = $2
WHERE auth_uuid = $1 AND revoked_at IS NULL;

-- name: TouchAuthByAuthUUID :exec
UPDATE auths
SET last_active_at = now()
WHERE auth_uuid = $1 AND revoked_at IS NULL;
