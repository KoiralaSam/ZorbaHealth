-- name: CountPatientCalls :one
SELECT COUNT(*)::int
FROM calls
WHERE patient_id = sqlc.arg(patient_id)::uuid;

-- name: GetPatientCallHistoryRows :many
SELECT
  ca.started_at AS started_at,
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
WHERE ca.patient_id = sqlc.arg(patient_id)::uuid
  AND ca.started_at IS NOT NULL
ORDER BY ca.started_at DESC NULLS LAST
LIMIT sqlc.arg(result_limit);
