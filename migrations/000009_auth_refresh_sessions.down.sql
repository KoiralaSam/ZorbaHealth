DROP TABLE IF EXISTS refresh_tokens;
DROP INDEX IF EXISTS auths_auth_uuid_unique_idx;
ALTER TABLE auths
    DROP COLUMN IF EXISTS client_kind,
    DROP COLUMN IF EXISTS last_active_at,
    DROP COLUMN IF EXISTS revoked_reason,
    DROP COLUMN IF EXISTS revoked_at;
