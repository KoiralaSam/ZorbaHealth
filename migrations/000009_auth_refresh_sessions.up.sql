CREATE UNIQUE INDEX IF NOT EXISTS auths_auth_uuid_unique_idx ON auths (auth_uuid);

ALTER TABLE auths
    ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS revoked_reason TEXT,
    ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ DEFAULT now(),
    ADD COLUMN IF NOT EXISTS client_kind TEXT;

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    auth_uuid UUID NOT NULL REFERENCES auths (auth_uuid) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    actor_type TEXT NOT NULL,
    token_hash BYTEA NOT NULL,
    generation INT NOT NULL DEFAULT 1,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS refresh_tokens_token_hash_idx ON refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS refresh_tokens_auth_uuid_idx ON refresh_tokens (auth_uuid, generation DESC);
CREATE INDEX IF NOT EXISTS auths_revoked_idx ON auths (user_id) WHERE revoked_at IS NULL;
