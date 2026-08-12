-- Enable btree_gist for exclusion constraints on (uuid, tstzrange).
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS staff_availability_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id uuid NOT NULL REFERENCES hospital_staff (id) ON DELETE CASCADE,
    hospital_id uuid NOT NULL REFERENCES hospitals (id) ON DELETE CASCADE,
    weekday smallint NOT NULL CHECK (weekday >= 0 AND weekday <= 6),
    start_time_local time NOT NULL,
    end_time_local time NOT NULL,
    slot_duration_minutes int NOT NULL CHECK (slot_duration_minutes > 0 AND slot_duration_minutes <= 480),
    timezone text NOT NULL DEFAULT 'UTC',
    effective_from date NOT NULL DEFAULT CURRENT_DATE,
    effective_until date,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (end_time_local > start_time_local),
    CHECK (effective_until IS NULL OR effective_until >= effective_from)
);

CREATE INDEX IF NOT EXISTS staff_availability_rules_staff_hospital_idx
    ON staff_availability_rules (staff_id, hospital_id);

CREATE TABLE IF NOT EXISTS staff_availability_exceptions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id uuid NOT NULL REFERENCES hospital_staff (id) ON DELETE CASCADE,
    hospital_id uuid NOT NULL REFERENCES hospitals (id) ON DELETE CASCADE,
    time_range tstzrange NOT NULL,
    reason text NOT NULL DEFAULT '',
    is_available boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (NOT isempty(time_range))
);

CREATE INDEX IF NOT EXISTS staff_availability_exceptions_staff_range_idx
    ON staff_availability_exceptions USING gist (staff_id, time_range);

CREATE TABLE IF NOT EXISTS appointments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id uuid NOT NULL REFERENCES patients (id),
    staff_id uuid NOT NULL REFERENCES hospital_staff (id),
    hospital_id uuid NOT NULL REFERENCES hospitals (id),
    time_range tstzrange NOT NULL,
    duration_minutes int NOT NULL CHECK (duration_minutes > 0 AND duration_minutes <= 480),
    timezone text NOT NULL DEFAULT 'UTC',
    type text NOT NULL DEFAULT 'video' CHECK (type IN ('video', 'in_person')),
    status text NOT NULL DEFAULT 'booked' CHECK (status IN ('booked', 'cancelled', 'completed', 'no_show')),
    channel text NOT NULL DEFAULT 'portal' CHECK (channel IN ('voice', 'portal', 'mobile', 'dashboard')),
    title text NOT NULL DEFAULT 'Zorba Health appointment',
    notes text,
    correlation_id uuid NOT NULL,
    voice_session_id text,
    booked_by_actor_type text NOT NULL,
    booked_by_actor_id text NOT NULL,
    join_url text,
    livekit_room_name text,
    livekit_room_sid text,
    patient_token text,
    staff_token text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (NOT isempty(time_range)),
    EXCLUDE USING gist (
        staff_id WITH =,
        time_range WITH &&
    ) WHERE (status = 'booked')
);

CREATE INDEX IF NOT EXISTS appointments_patient_starts_idx
    ON appointments (patient_id, lower(time_range) DESC);
CREATE INDEX IF NOT EXISTS appointments_staff_starts_idx
    ON appointments (staff_id, lower(time_range) DESC);
CREATE INDEX IF NOT EXISTS appointments_hospital_starts_idx
    ON appointments (hospital_id, lower(time_range) DESC);
CREATE INDEX IF NOT EXISTS appointments_correlation_idx
    ON appointments (correlation_id);
CREATE INDEX IF NOT EXISTS appointments_status_idx
    ON appointments (status);

INSERT INTO audit.audit_event_types (event_type, retention_class, description) VALUES
    ('APPOINTMENT_BOOKED', 'clinical_ops', 'Appointment booked against a staff availability slot.'),
    ('APPOINTMENT_RESCHEDULED', 'clinical_ops', 'Appointment rescheduled to a new slot.'),
    ('APPOINTMENT_CANCELLED', 'clinical_ops', 'Appointment cancelled.'),
    ('APPOINTMENT_BOOK_DENIED', 'clinical_ops', 'Appointment booking denied (consent, auth, slot conflict, or validation).'),
    ('AVAILABILITY_UPDATED', 'clinical_ops', 'Staff availability rules or exceptions updated.')
ON CONFLICT (event_type) DO NOTHING;
