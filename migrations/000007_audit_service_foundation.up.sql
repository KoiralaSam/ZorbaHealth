CREATE SCHEMA IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.audit_event_types (
    event_type TEXT PRIMARY KEY,
    retention_class TEXT NOT NULL,
    description TEXT NOT NULL
);

INSERT INTO audit.audit_event_types (event_type, retention_class, description) VALUES
    ('PATIENT_CREATED', 'compliance', 'A patient record was created.'),
    ('PATIENT_VERIFIED', 'compliance', 'A patient completed verification.'),
    ('PATIENT_LOGIN', 'security', 'A patient login succeeded.'),
    ('PATIENT_LOGOUT', 'security', 'A patient logout completed.'),
    ('HEALTH_RECORD_CREATED', 'clinical', 'A health record or document was ingested.'),
    ('HEALTH_RECORD_VIEWED', 'clinical', 'A user viewed structured or raw health record data.'),
    ('HEALTH_RECORD_SEARCHED', 'clinical', 'A semantic or structured record search was executed.'),
    ('HEALTH_RECORD_SUMMARIZED', 'clinical', 'A record summary was generated.'),
    ('AI_TOOL_CALLED', 'ai', 'An AI-facing backend tool call started or completed.'),
    ('AI_RESPONSE_GENERATED', 'ai', 'The conversational AI produced or attempted a response.'),
    ('LOCATION_REQUESTED', 'operational', 'A caller or staff location lookup was requested.'),
    ('EMERGENCY_ESCALATION_TRIGGERED', 'safety', 'An emergency escalation path was triggered.'),
    ('NOTIFICATION_SENT', 'operational', 'A notification delivery attempt was made.'),
    ('CONSENT_GRANTED', 'compliance', 'A patient consent was granted.'),
    ('CONSENT_REVOKED', 'compliance', 'A patient consent was revoked.'),
    ('TRANSLATION_REQUESTED', 'operational', 'A translation request was made.')
ON CONFLICT (event_type) DO NOTHING;

CREATE TABLE IF NOT EXISTS audit.audit_events (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL REFERENCES audit.audit_event_types(event_type),
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    patient_id TEXT,
    service_name TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    event_timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    request_id TEXT,
    correlation_id TEXT,
    ip_address TEXT,
    tool_name TEXT,
    model_name TEXT,
    provider_name TEXT,
    success_status BOOLEAN NOT NULL DEFAULT true,
    failure_reason TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_events_patient_idx
    ON audit.audit_events (patient_id, event_timestamp DESC);
CREATE INDEX IF NOT EXISTS audit_events_correlation_idx
    ON audit.audit_events (correlation_id, event_timestamp DESC);
CREATE INDEX IF NOT EXISTS audit_events_type_idx
    ON audit.audit_events (event_type, event_timestamp DESC);
CREATE INDEX IF NOT EXISTS audit_events_actor_idx
    ON audit.audit_events (actor_type, actor_id, event_timestamp DESC);

CREATE TABLE IF NOT EXISTS audit.consents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id TEXT NOT NULL,
    consent_type TEXT NOT NULL,
    granted_by TEXT NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    scope TEXT NOT NULL DEFAULT '',
    expiration_time TIMESTAMPTZ,
    source TEXT NOT NULL DEFAULT 'system',
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS audit_consents_patient_idx
    ON audit.consents (patient_id, consent_type, granted_at DESC);
CREATE INDEX IF NOT EXISTS audit_consents_active_idx
    ON audit.consents (patient_id, consent_type, scope)
    WHERE revoked_at IS NULL;
