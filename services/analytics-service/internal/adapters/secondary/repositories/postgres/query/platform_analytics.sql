-- name: GetPlatformSummary :one
WITH platform_calls AS (
  SELECT
    ca.patient_id,
    ca.started_at,
    CASE
      WHEN ca.ended_at IS NOT NULL AND ca.started_at IS NOT NULL AND ca.ended_at >= ca.started_at
        THEN EXTRACT(EPOCH FROM (ca.ended_at - ca.started_at))::double precision
      ELSE NULL::double precision
    END AS duration_seconds,
    EXISTS (
      SELECT 1
      FROM mcp_audit_log al
      WHERE al.session_id = ca.livekit_room_id
        AND al.tool = 'trigger_emergency'
        AND al.outcome = 'success'
    ) AS had_emergency
  FROM calls ca
  WHERE ca.started_at IS NOT NULL
),
active_hospital_ids AS (
  SELECT DISTINCT c.hospital_id
  FROM calls ca
  INNER JOIN patient_hospital_consents c
    ON c.patient_id = ca.patient_id
   AND c.revoked_at IS NULL
  WHERE ca.started_at >= now() - INTERVAL '7 days'
)
SELECT
  (
    SELECT COUNT(*)::int
    FROM hospitals h
    WHERE h.active = true
  ) AS total_hospitals,
  (
    SELECT COUNT(*)::int
    FROM patients
  ) AS total_patients,
  (
    SELECT COUNT(*)::int
    FROM platform_calls pc
    WHERE pc.started_at >= now() - INTERVAL '30 days'
  ) AS total_calls_30d,
  (
    SELECT COUNT(*)::int
    FROM platform_calls pc
    WHERE pc.started_at >= now() - INTERVAL '30 days'
      AND pc.had_emergency
  ) AS total_emergencies_30d,
  (
    SELECT COALESCE(AVG(pc.duration_seconds), 0)::double precision
    FROM platform_calls pc
    WHERE pc.started_at >= now() - INTERVAL '30 days'
  ) AS avg_call_duration_sec,
  (
    SELECT COUNT(*)::int
    FROM active_hospital_ids
  ) AS active_hospitals_7d;

-- name: GetPlatformHospitalBreakdown :many
WITH patient_counts AS (
  SELECT
    c.hospital_id,
    COUNT(DISTINCT c.patient_id)::int AS patient_count
  FROM patient_hospital_consents c
  WHERE c.revoked_at IS NULL
  GROUP BY c.hospital_id
),
call_counts AS (
  SELECT
    c.hospital_id,
    COUNT(*)::int AS call_count,
    COUNT(*) FILTER (
      WHERE EXISTS (
        SELECT 1
        FROM mcp_audit_log al
        WHERE al.session_id = ca.livekit_room_id
          AND al.tool = 'trigger_emergency'
          AND al.outcome = 'success'
      )
    )::int AS emergency_count
  FROM calls ca
  INNER JOIN patient_hospital_consents c
    ON c.patient_id = ca.patient_id
   AND c.revoked_at IS NULL
  WHERE ca.started_at >= now() - INTERVAL '30 days'
  GROUP BY c.hospital_id
)
SELECT
  h.id::text AS hospital_id,
  h.name AS hospital_name,
  COALESCE(pc.patient_count, 0)::int AS patient_count,
  COALESCE(cc.call_count, 0)::int AS call_count,
  COALESCE(cc.emergency_count, 0)::int AS emergency_count
FROM hospitals h
LEFT JOIN patient_counts pc ON pc.hospital_id = h.id
LEFT JOIN call_counts cc ON cc.hospital_id = h.id
WHERE h.active = true
ORDER BY
  CASE WHEN sqlc.arg(sort_by)::text = 'calls' THEN COALESCE(cc.call_count, 0) END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'patients' THEN COALESCE(pc.patient_count, 0) END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'emergencies' THEN COALESCE(cc.emergency_count, 0) END DESC,
  h.name ASC
LIMIT sqlc.arg(result_limit);
