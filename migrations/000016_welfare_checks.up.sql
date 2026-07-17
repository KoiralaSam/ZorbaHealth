CREATE TABLE IF NOT EXISTS welfare_check_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id uuid NOT NULL REFERENCES patients (id),
    scheduled_at timestamptz NOT NULL,
    timezone text NOT NULL DEFAULT 'UTC',
    reason_code text NOT NULL CHECK (reason_code IN (
        'medication_reminder',
        'mental_wellbeing',
        'daily_checkup',
        'symptom_follow_up',
        'care_plan_reminder',
        'other'
    )),
    reason_detail text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'scheduled' CHECK (status IN (
        'scheduled', 'cancelled', 'completed', 'missed', 'failed'
    )),
    recurrence_rule text,
    recurrence_starts_at timestamptz,
    recurrence_ends_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    cancelled_at timestamptz
);

CREATE TABLE IF NOT EXISTS welfare_check_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id uuid NOT NULL REFERENCES welfare_check_requests (id) ON DELETE CASCADE,
    patient_id uuid NOT NULL REFERENCES patients (id),
    scheduled_at timestamptz NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'claimed', 'dispatched', 'answered', 'missed', 'completed', 'failed', 'cancelled'
    )),
    attempts int NOT NULL DEFAULT 0,
    last_attempt_at timestamptz,
    next_attempt_at timestamptz,
    livekit_room_name text,
    livekit_room_sid text,
    livekit_dispatch_id text,
    livekit_sip_call_id text,
    failure_reason text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS welfare_check_requests_patient_scheduled_idx
    ON welfare_check_requests (patient_id, scheduled_at DESC);

CREATE INDEX IF NOT EXISTS welfare_check_runs_due_idx
    ON welfare_check_runs (status, next_attempt_at, scheduled_at);

CREATE INDEX IF NOT EXISTS welfare_check_runs_request_idx
    ON welfare_check_runs (request_id);

CREATE UNIQUE INDEX IF NOT EXISTS welfare_check_runs_livekit_room_name_uidx
    ON welfare_check_runs (livekit_room_name)
    WHERE livekit_room_name IS NOT NULL AND livekit_room_name <> '';

INSERT INTO audit.audit_event_types (event_type, retention_class, description) VALUES
    ('WELFARE_CHECK_REQUESTED', 'clinical_ops', 'Patient requested a scheduled welfare check.'),
    ('WELFARE_CHECK_CANCELLED', 'clinical_ops', 'Patient cancelled a scheduled welfare check.'),
    ('WELFARE_CHECK_DISPATCHED', 'clinical_ops', 'Scheduled welfare check outbound call dispatched.'),
    ('WELFARE_CHECK_ANSWERED', 'clinical_ops', 'Scheduled welfare check outbound call answered.'),
    ('WELFARE_CHECK_MISSED', 'clinical_ops', 'Scheduled welfare check outbound call was missed.'),
    ('WELFARE_CHECK_FAILED', 'clinical_ops', 'Scheduled welfare check dispatch failed.'),
    ('WELFARE_CHECK_COMPLETED', 'clinical_ops', 'Scheduled welfare check completed.'),
    ('WELFARE_CHECK_RECORD_CONTEXT_ACCESSED', 'clinical_ops', 'Scheduled welfare check used pre-authorized patient record context.')
ON CONFLICT (event_type) DO NOTHING;
