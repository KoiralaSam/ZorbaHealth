-- HOSPITAL + PLATFORM ANALYTICS VIEWS
--
-- NOTE: these views depend on application tables that may not exist in early dev environments.
-- We guard view creation to avoid leaving the DB in a dirty migration state.

DO $$
BEGIN
    -- Daily call volume per hospital (last 90 days)
    IF to_regclass('public.patient_hospital_consents') IS NOT NULL
       AND to_regclass('public.patients') IS NOT NULL
       AND to_regclass('public.calls') IS NOT NULL
       AND to_regclass('public.mcp_audit_log') IS NOT NULL
       AND to_regclass('public.mv_hospital_call_daily') IS NULL THEN
        EXECUTE $sql$
        CREATE MATERIALIZED VIEW mv_hospital_call_daily AS
        SELECT
            c.hospital_id,
            DATE(ca.started_at) AS call_date,
            COUNT(*) AS total_calls,
            COUNT(*) FILTER (WHERE ca.status = 'ended') AS completed_calls,
            AVG(EXTRACT(EPOCH FROM (ca.ended_at - ca.started_at))) AS avg_duration_seconds,
            COUNT(*) FILTER (
                WHERE al.tool = 'trigger_emergency'
                  AND al.outcome = 'success'
            ) AS emergency_calls
        FROM patient_hospital_consents c
        JOIN patients p ON p.id = c.patient_id
        JOIN calls ca ON ca.patient_id = p.id
        LEFT JOIN mcp_audit_log al
               ON al.session_id = ca.livekit_room_id
              AND al.tool = 'trigger_emergency'
        WHERE c.revoked_at IS NULL
          AND ca.started_at >= now() - interval '90 days'
        GROUP BY c.hospital_id, DATE(ca.started_at)
        WITH DATA;
        $sql$;

        IF to_regclass('public.idx_mv_hospital_call_daily_hospital_date') IS NULL THEN
            EXECUTE $sql$
            CREATE UNIQUE INDEX idx_mv_hospital_call_daily_hospital_date
                ON mv_hospital_call_daily (hospital_id, call_date);
            $sql$;
        END IF;
    END IF;

    -- Top conditions mentioned per hospital (derived from FHIR resources)
    IF to_regclass('public.patient_hospital_consents') IS NOT NULL
       AND to_regclass('public.fhir_resources') IS NOT NULL
       AND to_regclass('public.mv_hospital_top_conditions') IS NULL THEN
        EXECUTE $sql$
        CREATE MATERIALIZED VIEW mv_hospital_top_conditions AS
        SELECT
            c.hospital_id,
            f.resource_json->>'resourceType' AS resource_type,
            f.resource_json->'code'->>'text' AS condition_name,
            COUNT(*) AS patient_count
        FROM patient_hospital_consents c
        JOIN fhir_resources f ON f.patient_id = c.patient_id
        WHERE c.revoked_at IS NULL
          AND f.resource_type = 'Condition'
          AND f.resource_json->'clinicalStatus'->'coding'->0->>'code' = 'active'
        GROUP BY
            c.hospital_id,
            f.resource_json->>'resourceType',
            f.resource_json->'code'->>'text'
        ORDER BY patient_count DESC
        WITH DATA;
        $sql$;

        IF to_regclass('public.idx_mv_hospital_top_conditions_hospital_condition') IS NULL THEN
            EXECUTE $sql$
            CREATE UNIQUE INDEX idx_mv_hospital_top_conditions_hospital_condition
                ON mv_hospital_top_conditions (hospital_id, condition_name);
            $sql$;
        END IF;
    END IF;

    -- Tool usage breakdown per hospital
    IF to_regclass('public.mcp_audit_log') IS NOT NULL
       AND to_regclass('public.v_hospital_tool_usage') IS NULL THEN
        EXECUTE $sql$
        CREATE VIEW v_hospital_tool_usage AS
        SELECT
            al.hospital_id,
            al.tool,
            al.outcome,
            COUNT(*) AS call_count,
            DATE_TRUNC('day', al.timestamp) AS day
        FROM mcp_audit_log al
        WHERE al.actor_type IN ('patient', 'staff')
        GROUP BY al.hospital_id, al.tool, al.outcome, DATE_TRUNC('day', al.timestamp);
        $sql$;
    END IF;

    -- System-wide daily metrics (last 365 days)
    IF to_regclass('public.calls') IS NOT NULL
       AND to_regclass('public.patient_hospital_consents') IS NOT NULL
       AND to_regclass('public.mcp_audit_log') IS NOT NULL
       AND to_regclass('public.mv_platform_daily') IS NULL THEN
        EXECUTE $sql$
        CREATE MATERIALIZED VIEW mv_platform_daily AS
        SELECT
            DATE(ca.started_at) AS call_date,
            COUNT(DISTINCT ca.patient_id) AS unique_patients,
            COUNT(*) AS total_calls,
            COUNT(*) FILTER (
                WHERE al.tool = 'trigger_emergency'
                  AND al.outcome = 'success'
            ) AS emergencies,
            AVG(EXTRACT(EPOCH FROM (ca.ended_at - ca.started_at))) AS avg_duration_seconds,
            COUNT(DISTINCT c.hospital_id) AS active_hospitals
        FROM calls ca
        LEFT JOIN patient_hospital_consents c ON c.patient_id = ca.patient_id
        LEFT JOIN mcp_audit_log al ON al.session_id = ca.livekit_room_id
        WHERE ca.started_at >= now() - interval '365 days'
        GROUP BY DATE(ca.started_at)
        WITH DATA;
        $sql$;

        IF to_regclass('public.idx_mv_platform_daily_call_date') IS NULL THEN
            EXECUTE $sql$
            CREATE UNIQUE INDEX idx_mv_platform_daily_call_date
                ON mv_platform_daily (call_date);
            $sql$;
        END IF;
    END IF;
END;
$$;

-- REFRESH FUNCTION
-- Call this from analytics-service on a schedule (or use pg_cron)
CREATE OR REPLACE FUNCTION refresh_analytics_views()
RETURNS void AS $$
BEGIN
    IF to_regclass('public.mv_hospital_call_daily') IS NOT NULL THEN
        REFRESH MATERIALIZED VIEW CONCURRENTLY mv_hospital_call_daily;
    END IF;
    IF to_regclass('public.mv_hospital_top_conditions') IS NOT NULL THEN
        REFRESH MATERIALIZED VIEW CONCURRENTLY mv_hospital_top_conditions;
    END IF;
    IF to_regclass('public.mv_platform_daily') IS NOT NULL THEN
        REFRESH MATERIALIZED VIEW CONCURRENTLY mv_platform_daily;
    END IF;
END;
$$ LANGUAGE plpgsql;
