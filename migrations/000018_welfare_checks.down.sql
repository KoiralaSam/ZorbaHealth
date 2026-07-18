DELETE FROM audit.audit_event_types
WHERE event_type IN (
    'WELFARE_CHECK_REQUESTED',
    'WELFARE_CHECK_CANCELLED',
    'WELFARE_CHECK_DISPATCHED',
    'WELFARE_CHECK_ANSWERED',
    'WELFARE_CHECK_MISSED',
    'WELFARE_CHECK_FAILED',
    'WELFARE_CHECK_COMPLETED',
    'WELFARE_CHECK_RECORD_CONTEXT_ACCESSED'
);

DROP INDEX IF EXISTS welfare_check_runs_livekit_room_name_uidx;
DROP INDEX IF EXISTS welfare_check_runs_request_idx;
DROP INDEX IF EXISTS welfare_check_runs_due_idx;
DROP INDEX IF EXISTS welfare_check_requests_patient_scheduled_idx;

DROP TABLE IF EXISTS welfare_check_runs;
DROP TABLE IF EXISTS welfare_check_requests;
