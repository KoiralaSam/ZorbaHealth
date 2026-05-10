-- name: GetHospitalSummary :one
WITH consented_patients AS (
  SELECT c.patient_id
  FROM patient_hospital_consents c
  WHERE c.hospital_id = sqlc.arg(hospital_id)::uuid
    AND c.revoked_at IS NULL
),
hospital_calls AS (
  SELECT
    ca.patient_id,
    ca.started_at,
    ca.started_at::date AS call_date,
    ca.status,
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
  INNER JOIN consented_patients cp ON cp.patient_id = ca.patient_id
  WHERE ca.started_at IS NOT NULL
)
SELECT
  sqlc.arg(hospital_id)::text AS hospital_id,
  (SELECT COUNT(*)::int FROM consented_patients) AS total_consented_patients,
  (
    SELECT COUNT(*)::int
    FROM hospital_calls hc
    WHERE hc.started_at >= now() - INTERVAL '30 days'
  ) AS total_calls_30d,
  (
    SELECT COUNT(*)::int
    FROM hospital_calls hc
    WHERE hc.started_at >= now() - INTERVAL '30 days'
      AND hc.had_emergency
  ) AS emergency_events_30d,
  (
    SELECT COALESCE(AVG(hc.duration_seconds), 0)::double precision
    FROM hospital_calls hc
    WHERE hc.started_at >= now() - INTERVAL '30 days'
  ) AS avg_call_duration_seconds,
  (
    SELECT COUNT(rc.id)::int
    FROM record_chunks rc
    INNER JOIN consented_patients cp ON cp.patient_id = rc.patient_id
  ) AS records_indexed,
  (
    SELECT COUNT(DISTINCT ca.patient_id)::int
    FROM calls ca
    INNER JOIN consented_patients cp ON cp.patient_id = ca.patient_id
    WHERE ca.started_at >= now() - INTERVAL '7 days'
  ) AS active_patients_7d;

-- name: GetHospitalCallVolume :many
WITH consented_patients AS (
  SELECT c.patient_id
  FROM patient_hospital_consents c
  WHERE c.hospital_id = sqlc.arg(hospital_id)::uuid
    AND c.revoked_at IS NULL
),
hospital_calls AS (
  SELECT
    ca.started_at,
    CASE
      WHEN ca.ended_at IS NOT NULL AND ca.started_at IS NOT NULL AND ca.ended_at >= ca.started_at
        THEN EXTRACT(EPOCH FROM (ca.ended_at - ca.started_at))::double precision
      ELSE NULL::double precision
    END AS duration_seconds,
    (ca.status = 'ended') AS completed,
    EXISTS (
      SELECT 1
      FROM mcp_audit_log al
      WHERE al.session_id = ca.livekit_room_id
        AND al.tool = 'trigger_emergency'
        AND al.outcome = 'success'
    ) AS had_emergency
  FROM calls ca
  INNER JOIN consented_patients cp ON cp.patient_id = ca.patient_id
  WHERE ca.started_at IS NOT NULL
)
SELECT
  CASE
    WHEN sqlc.arg(granularity)::text = 'week' THEN date_trunc('week', hc.started_at)
    WHEN sqlc.arg(granularity)::text = 'month' THEN date_trunc('month', hc.started_at)
    ELSE date_trunc('day', hc.started_at)
  END AS date,
  COUNT(*)::int AS total_calls,
  COUNT(*) FILTER (WHERE hc.completed)::int AS completed_calls,
  COUNT(*) FILTER (WHERE hc.had_emergency)::int AS emergency_calls,
  COALESCE(AVG(hc.duration_seconds), 0)::double precision AS avg_duration_sec
FROM hospital_calls hc
WHERE hc.started_at >= sqlc.arg(from_date)::timestamp
  AND hc.started_at < sqlc.arg(to_date)::timestamp + INTERVAL '1 day'
  AND sqlc.arg(granularity)::text IN ('day', 'week', 'month')
GROUP BY 1
ORDER BY 1 ASC;

-- name: GetHospitalTopConditions :many
WITH total_patients AS (
  SELECT COUNT(*)::int AS total
  FROM patient_hospital_consents c
  WHERE c.hospital_id = sqlc.arg(hospital_id)::uuid
    AND c.revoked_at IS NULL
)
SELECT
  f.resource_json->'code'->>'text' AS condition_name,
  COUNT(DISTINCT c.patient_id)::int AS patient_count,
  CASE
    WHEN tp.total > 0 THEN (COUNT(DISTINCT c.patient_id)::double precision / tp.total::double precision) * 100
    ELSE 0::double precision
  END AS percentage
FROM patient_hospital_consents c
INNER JOIN fhir_resources f ON f.patient_id = c.patient_id
CROSS JOIN total_patients tp
WHERE c.hospital_id = sqlc.arg(hospital_id)::uuid
  AND c.revoked_at IS NULL
  AND f.resource_type = 'Condition'
  AND f.resource_json->'clinicalStatus'->'coding'->0->>'code' = 'active'
GROUP BY f.resource_json->'code'->>'text', tp.total
ORDER BY patient_count DESC, condition_name ASC
LIMIT sqlc.arg(result_limit);

-- name: GetHospitalToolUsage :many
SELECT
  al.tool,
  COUNT(*) FILTER (WHERE al.outcome = 'success')::int AS success_count,
  COUNT(*) FILTER (WHERE al.outcome = 'error')::int AS error_count,
  COUNT(*) FILTER (WHERE al.outcome IN ('forbidden', 'consent-denied'))::int AS denied_count,
  CASE
    WHEN COUNT(*) > 0
      THEN (
        COUNT(*) FILTER (WHERE al.outcome = 'success')::double precision
        / COUNT(*)::double precision
      ) * 100
    ELSE 0::double precision
  END AS success_rate
FROM mcp_audit_log al
WHERE al.hospital_id = sqlc.arg(hospital_id)::text
  AND al.timestamp >= sqlc.arg(from_time)
GROUP BY al.tool
ORDER BY success_count DESC, al.tool ASC;

-- name: GetHospitalRecentActivity :many
SELECT
  al.timestamp,
  al.tool,
  al.actor_type,
  al.outcome,
  al.session_id
FROM mcp_audit_log al
WHERE al.hospital_id = sqlc.arg(hospital_id)::text
ORDER BY al.timestamp DESC
LIMIT sqlc.arg(result_limit);
