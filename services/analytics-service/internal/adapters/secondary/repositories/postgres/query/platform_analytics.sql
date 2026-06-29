-- name: GetPlatformSummary :one
WITH
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
    FROM analytics.platform_daily_mv pc
    WHERE pc.call_date >= CURRENT_DATE - 30
  ) AS total_calls_30d,
  (
    SELECT COALESCE(SUM(pc.emergencies), 0)::int
    FROM analytics.platform_daily_mv pc
    WHERE pc.call_date >= CURRENT_DATE - 30
  ) AS total_emergencies_30d,
  (
    SELECT COALESCE(AVG(pc.avg_duration_seconds), 0)::double precision
    FROM analytics.platform_daily_mv pc
    WHERE pc.call_date >= CURRENT_DATE - 30
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
    hospital_id,
    COALESCE(SUM(total_calls), 0)::int AS call_count,
    COALESCE(SUM(emergency_calls), 0)::int AS emergency_count
  FROM analytics.hospital_call_daily_mv
  WHERE call_date >= CURRENT_DATE - 30
  GROUP BY hospital_id
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
