-- name: GetHospitalSummary :one
WITH consented_patients AS (
  SELECT c.patient_id
  FROM patient_hospital_consents c
  WHERE c.hospital_id = sqlc.arg(hospital_id)::uuid
    AND c.revoked_at IS NULL
)
SELECT
  sqlc.arg(hospital_id)::text AS hospital_id,
  (SELECT COUNT(*)::int FROM consented_patients) AS total_consented_patients,
  (
    SELECT COALESCE(SUM(hc.total_calls), 0)::int
    FROM analytics.hospital_call_daily_mv hc
    WHERE hc.hospital_id = sqlc.arg(hospital_id)::uuid
      AND hc.call_date >= CURRENT_DATE - 30
  ) AS total_calls_30d,
  (
    SELECT COALESCE(SUM(hc.emergency_calls), 0)::int
    FROM analytics.hospital_call_daily_mv hc
    WHERE hc.hospital_id = sqlc.arg(hospital_id)::uuid
      AND hc.call_date >= CURRENT_DATE - 30
  ) AS emergency_events_30d,
  (
    SELECT COALESCE(AVG(hc.avg_duration_seconds), 0)::double precision
    FROM analytics.hospital_call_daily_mv hc
    WHERE hc.hospital_id = sqlc.arg(hospital_id)::uuid
      AND hc.call_date >= CURRENT_DATE - 30
  ) AS avg_call_duration_seconds,
  (
    SELECT COUNT(rc.chunk_id)::int
    FROM records.record_chunks rc
    INNER JOIN consented_patients cp ON cp.patient_id = rc.patient_id
  ) AS records_indexed,
  (
    SELECT COUNT(DISTINCT ca.patient_id)::int
    FROM calls ca
    INNER JOIN consented_patients cp ON cp.patient_id = ca.patient_id
    WHERE ca.started_at >= now() - INTERVAL '7 days'
  ) AS active_patients_7d;

-- name: GetHospitalCallVolume :many
SELECT
  CASE
    WHEN sqlc.arg(granularity)::text = 'week' THEN date_trunc('week', hc.call_date::timestamp)
    WHEN sqlc.arg(granularity)::text = 'month' THEN date_trunc('month', hc.call_date::timestamp)
    ELSE date_trunc('day', hc.call_date::timestamp)
  END AS date,
  SUM(hc.total_calls)::int AS total_calls,
  SUM(hc.completed_calls)::int AS completed_calls,
  SUM(hc.emergency_calls)::int AS emergency_calls,
  COALESCE(AVG(hc.avg_duration_seconds), 0)::double precision AS avg_duration_sec
FROM analytics.hospital_call_daily_mv hc
WHERE hc.hospital_id = sqlc.arg(hospital_id)::uuid
  AND hc.call_date >= sqlc.arg(from_date)::date
  AND hc.call_date < sqlc.arg(to_date)::date + INTERVAL '1 day'
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
  f.condition_name,
  f.patient_count,
  CASE
    WHEN tp.total > 0 THEN (f.patient_count::double precision / tp.total::double precision) * 100
    ELSE 0::double precision
  END AS percentage
FROM analytics.hospital_top_conditions_mv f
CROSS JOIN total_patients tp
WHERE f.hospital_id = sqlc.arg(hospital_id)::uuid
GROUP BY f.condition_name, f.patient_count, tp.total
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
FROM analytics.hospital_tool_usage_v al
WHERE al.hospital_id = sqlc.arg(hospital_id)::text
  AND al.day >= sqlc.arg(from_time)::timestamp
GROUP BY al.tool
ORDER BY success_count DESC, al.tool ASC;

-- name: GetHospitalRecentActivity :many
SELECT
  al.timestamp,
  al.tool,
  al.actor_type,
  al.outcome,
  al.session_id
FROM analytics.hospital_recent_activity_v al
WHERE al.hospital_id = sqlc.arg(hospital_id)::text
ORDER BY al.timestamp DESC
LIMIT sqlc.arg(result_limit);
