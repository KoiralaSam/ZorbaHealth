-- Best-effort reverse of 000017. Prefer rolling back with 000016 when possible.
DROP INDEX IF EXISTS welfare_check_runs_livekit_room_name_uidx;

ALTER TABLE welfare_check_requests
    DROP CONSTRAINT IF EXISTS welfare_check_requests_status_check;

-- Keep missed out of the older CHECK only when no missed rows exist.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM welfare_check_requests WHERE status = 'missed'
    ) THEN
        ALTER TABLE welfare_check_requests
            ADD CONSTRAINT welfare_check_requests_status_check
            CHECK (status IN ('scheduled', 'cancelled', 'completed', 'failed'));
    ELSE
        ALTER TABLE welfare_check_requests
            ADD CONSTRAINT welfare_check_requests_status_check
            CHECK (status IN ('scheduled', 'cancelled', 'completed', 'missed', 'failed'));
    END IF;
END $$;
