-- name: InsertRefreshToken :one
INSERT INTO refresh_tokens (auth_uuid, user_id, actor_type, token_hash, generation, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, auth_uuid, user_id, actor_type, token_hash, generation, expires_at, used_at, revoked_at, created_at;

-- name: GetRefreshTokenByHash :one
SELECT id, auth_uuid, user_id, actor_type, token_hash, generation, expires_at, used_at, revoked_at, created_at
FROM refresh_tokens
WHERE token_hash = $1
LIMIT 1;

-- name: GetRefreshTokenByHashForUpdate :one
SELECT id, auth_uuid, user_id, actor_type, token_hash, generation, expires_at, used_at, revoked_at, created_at
FROM refresh_tokens
WHERE token_hash = $1
LIMIT 1
FOR UPDATE;

-- name: MarkRefreshTokenUsed :exec
UPDATE refresh_tokens
SET used_at = now()
WHERE id = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE id = $1;

-- name: RevokeAllRefreshTokensForAuthUUID :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE auth_uuid = $1 AND revoked_at IS NULL;

-- name: GetMaxGenerationForAuthUUID :one
SELECT COALESCE(MAX(generation), 0)::int AS max_generation
FROM refresh_tokens
WHERE auth_uuid = $1;
