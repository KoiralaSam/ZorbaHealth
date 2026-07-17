-- Compatibility migration for environments that applied an earlier 000016 without
-- the full status CHECK set or unique room-name index.
ALTER TABLE welfare_check_requests
    DROP CONSTRAINT IF EXISTS welfare_check_requests_status_check;

ALTER TABLE welfare_check_requests
    ADD CONSTRAINT welfare_check_requests_status_check
    CHECK (status IN ('scheduled', 'cancelled', 'completed', 'missed', 'failed'));

ALTER TABLE welfare_check_runs
    DROP CONSTRAINT IF EXISTS welfare_check_runs_status_check;

ALTER TABLE welfare_check_runs
    ADD CONSTRAINT welfare_check_runs_status_check
    CHECK (status IN ('pending', 'claimed', 'dispatched', 'answered', 'missed', 'completed', 'failed', 'cancelled'));

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
