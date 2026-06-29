CREATE SCHEMA IF NOT EXISTS analytics;

CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.hospital_call_daily_mv AS
SELECT
    c.hospital_id,
    DATE(ca.started_at) AS call_date,
    COUNT(*)::int AS total_calls,
    COUNT(*) FILTER (WHERE ca.status = 'ended')::int AS completed_calls,
    COALESCE(AVG(EXTRACT(EPOCH FROM (ca.ended_at - ca.started_at))), 0)::double precision AS avg_duration_seconds,
    COUNT(*) FILTER (
        WHERE EXISTS (
            SELECT 1
            FROM mcp_audit_log al
            WHERE al.session_id = ca.livekit_room_id
              AND al.tool = 'trigger_emergency'
              AND al.outcome = 'success'
        )
    )::int AS emergency_calls
FROM patient_hospital_consents c
JOIN calls ca ON ca.patient_id = c.patient_id
WHERE c.revoked_at IS NULL
  AND ca.started_at IS NOT NULL
GROUP BY c.hospital_id, DATE(ca.started_at)
WITH DATA;

CREATE UNIQUE INDEX IF NOT EXISTS analytics_hospital_call_daily_hospital_date_idx
    ON analytics.hospital_call_daily_mv (hospital_id, call_date);

CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.hospital_top_conditions_mv AS
SELECT
    c.hospital_id,
    COALESCE(r.display_text, r.resource_json->'code'->>'text', 'Unknown') AS condition_name,
    COUNT(DISTINCT c.patient_id)::int AS patient_count
FROM patient_hospital_consents c
JOIN records.fhir_resources r ON r.patient_id = c.patient_id
WHERE c.revoked_at IS NULL
  AND r.resource_type = 'Condition'
GROUP BY c.hospital_id, COALESCE(r.display_text, r.resource_json->'code'->>'text', 'Unknown')
WITH DATA;

CREATE UNIQUE INDEX IF NOT EXISTS analytics_hospital_top_conditions_idx
    ON analytics.hospital_top_conditions_mv (hospital_id, condition_name);

CREATE OR REPLACE VIEW analytics.hospital_tool_usage_v AS
SELECT
    al.hospital_id,
    al.tool,
    al.outcome,
    COUNT(*)::int AS call_count,
    DATE_TRUNC('day', al.timestamp) AS day
FROM mcp_audit_log al
WHERE al.actor_type IN ('patient', 'staff')
GROUP BY al.hospital_id, al.tool, al.outcome, DATE_TRUNC('day', al.timestamp);

CREATE OR REPLACE VIEW analytics.hospital_recent_activity_v AS
SELECT
    al.hospital_id,
    al.timestamp,
    al.tool,
    al.actor_type,
    al.outcome,
    al.session_id
FROM mcp_audit_log al
WHERE al.actor_type IN ('patient', 'staff');

CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.platform_daily_mv AS
SELECT
    DATE(ca.started_at) AS call_date,
    COUNT(DISTINCT ca.patient_id)::int AS unique_patients,
    COUNT(*)::int AS total_calls,
    COUNT(*) FILTER (
        WHERE EXISTS (
            SELECT 1
            FROM mcp_audit_log al
            WHERE al.session_id = ca.livekit_room_id
              AND al.tool = 'trigger_emergency'
              AND al.outcome = 'success'
        )
    )::int AS emergencies,
    COALESCE(AVG(EXTRACT(EPOCH FROM (ca.ended_at - ca.started_at))), 0)::double precision AS avg_duration_seconds,
    COUNT(DISTINCT c.hospital_id)::int AS active_hospitals
FROM calls ca
LEFT JOIN patient_hospital_consents c ON c.patient_id = ca.patient_id AND c.revoked_at IS NULL
WHERE ca.started_at IS NOT NULL
GROUP BY DATE(ca.started_at)
WITH DATA;

CREATE UNIQUE INDEX IF NOT EXISTS analytics_platform_daily_call_date_idx
    ON analytics.platform_daily_mv (call_date);

CREATE OR REPLACE VIEW analytics.patient_call_history_v AS
SELECT
    ca.patient_id,
    ca.started_at,
    CASE
        WHEN ca.ended_at IS NULL OR ca.started_at IS NULL OR ca.ended_at < ca.started_at THEN 'unknown'
        WHEN EXTRACT(EPOCH FROM (ca.ended_at - ca.started_at)) < 60
            THEN CONCAT(EXTRACT(EPOCH FROM (ca.ended_at - ca.started_at))::int, ' sec')
        ELSE CONCAT(
            FLOOR(EXTRACT(EPOCH FROM (ca.ended_at - ca.started_at)) / 60)::int,
            ' min ',
            MOD(EXTRACT(EPOCH FROM (ca.ended_at - ca.started_at))::int, 60),
            ' sec'
        )
    END AS duration,
    COALESCE(ca.summary, '') AS summary,
    EXISTS (
        SELECT 1
        FROM mcp_audit_log al
        WHERE al.session_id = ca.livekit_room_id
          AND al.tool = 'trigger_emergency'
          AND al.outcome = 'success'
    ) AS had_emergency
FROM calls ca
WHERE ca.started_at IS NOT NULL;

CREATE OR REPLACE FUNCTION analytics.refresh_safe_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW analytics.hospital_call_daily_mv;
    REFRESH MATERIALIZED VIEW analytics.hospital_top_conditions_mv;
    REFRESH MATERIALIZED VIEW analytics.platform_daily_mv;
END;
$$ LANGUAGE plpgsql;
