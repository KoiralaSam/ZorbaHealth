-- name: CountPatientCalls :one
SELECT COUNT(*)::int
FROM calls
WHERE patient_id = sqlc.arg(patient_id)::uuid;

-- name: GetPatientCallHistoryRows :many
SELECT
  started_at,
  duration,
  summary,
  had_emergency
FROM analytics.patient_call_history_v
WHERE patient_id = sqlc.arg(patient_id)::uuid
ORDER BY started_at DESC NULLS LAST
LIMIT sqlc.arg(result_limit);
