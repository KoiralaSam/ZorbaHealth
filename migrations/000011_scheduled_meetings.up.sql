CREATE TABLE IF NOT EXISTS scheduled_meetings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id uuid NOT NULL REFERENCES patients (id),
    staff_id uuid NOT NULL REFERENCES hospital_staff (id),
    hospital_id uuid NOT NULL REFERENCES hospitals (id),
    created_by_actor_type text NOT NULL,
    created_by_actor_id text NOT NULL,
    starts_at timestamptz NOT NULL,
    duration_minutes int NOT NULL CHECK (duration_minutes > 0 AND duration_minutes <= 480),
    timezone text NOT NULL,
    title text NOT NULL DEFAULT 'Zorba Health video visit',
    notes text,
    join_url text,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'scheduled', 'cancelled', 'failed')),
    correlation_id uuid NOT NULL,
    voice_session_id text,
    send_sms boolean NOT NULL DEFAULT false,
    channel text NOT NULL DEFAULT 'portal' CHECK (channel IN ('voice', 'portal', 'dashboard')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS scheduled_meetings_patient_starts_idx ON scheduled_meetings (patient_id, starts_at DESC);
CREATE INDEX IF NOT EXISTS scheduled_meetings_staff_starts_idx ON scheduled_meetings (staff_id, starts_at DESC);
CREATE INDEX IF NOT EXISTS scheduled_meetings_hospital_starts_idx ON scheduled_meetings (hospital_id, starts_at DESC);
CREATE INDEX IF NOT EXISTS scheduled_meetings_correlation_idx ON scheduled_meetings (correlation_id);

INSERT INTO audit.audit_event_types (event_type, retention_class, description) VALUES
    ('MEETING_REQUESTED', 'clinical_ops', 'Health staff video meeting requested and awaiting staff approval.'),
    ('MEETING_ACCEPTED', 'clinical_ops', 'Pending health staff video meeting accepted by staff.'),
    ('MEETING_RESCHEDULED', 'clinical_ops', 'Pending health staff video meeting rescheduled by staff.'),
    ('MEETING_SCHEDULED', 'clinical_ops', 'Health staff video meeting scheduled.'),
    ('MEETING_CANCELLED', 'clinical_ops', 'Scheduled health staff meeting cancelled.'),
    ('MEETING_SCHEDULE_DENIED', 'clinical_ops', 'Meeting schedule denied (consent, auth, or validation).')
ON CONFLICT (event_type) DO NOTHING;
