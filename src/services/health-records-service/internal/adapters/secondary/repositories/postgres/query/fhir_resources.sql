-- Structured FHIR R4 resources (JSONB)

-- name: UpsertFHIRResource :one
INSERT INTO fhir_resources (
  patient_id,
  resource_type,
  resource_id,
  source_system,
  resource_json,
  indexed_at
) VALUES (
  $1, $2, $3, $4, $5::jsonb, now()
)
ON CONFLICT (patient_id, resource_type, resource_id)
DO UPDATE SET
  source_system = EXCLUDED.source_system,
  resource_json = EXCLUDED.resource_json,
  indexed_at = now()
RETURNING *;

-- name: ListFHIRResourcesByType :many
SELECT
  resource_json::text AS resource_json
FROM fhir_resources
WHERE patient_id = $1
  AND resource_type = $2
ORDER BY indexed_at DESC
LIMIT $3 OFFSET $4;

-- name: ListFHIRResourcesByTypeAndStatus :many
SELECT
  resource_json::text AS resource_json
FROM fhir_resources
WHERE patient_id = $1
  AND resource_type = $2
  AND ($3 = '' OR resource_json->>'status' = $3)
ORDER BY indexed_at DESC
LIMIT $4 OFFSET $5;

-- name: CountFHIRResourcesByType :one
SELECT COUNT(*)
FROM fhir_resources
WHERE patient_id = $1
  AND resource_type = $2;

-- name: DeleteFHIRResourcesByPatientID :exec
DELETE FROM fhir_resources
WHERE patient_id = $1;

