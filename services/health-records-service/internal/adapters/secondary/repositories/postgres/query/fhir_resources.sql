-- Structured FHIR R4 resources (JSONB) in records schema

-- name: UpsertFHIRResource :one
INSERT INTO records.fhir_resources (
  patient_id,
  resource_type,
  resource_id,
  source_system,
  resource_json,
  display_text,
  clinical_status,
  effective_date,
  indexed_at
) VALUES (
  $1, $2, $3, $4, $5::jsonb, $6, $7, $8, now()
)
ON CONFLICT (patient_id, resource_type, resource_id)
DO UPDATE SET
  source_system = EXCLUDED.source_system,
  resource_json = EXCLUDED.resource_json,
  display_text = EXCLUDED.display_text,
  clinical_status = EXCLUDED.clinical_status,
  effective_date = EXCLUDED.effective_date,
  indexed_at = now()
RETURNING *;

-- name: ListFHIRResourcesByType :many
SELECT
  resource_json::text AS resource_json
FROM records.fhir_resources
WHERE patient_id = $1
  AND resource_type = $2
ORDER BY indexed_at DESC
LIMIT $3 OFFSET $4;

-- name: ListFHIRResourcesByTypeAndStatus :many
SELECT
  resource_json::text AS resource_json
FROM records.fhir_resources
WHERE patient_id = $1
  AND resource_type = $2
  AND ($3 = '' OR resource_json->>'status' = $3 OR clinical_status = $3)
ORDER BY indexed_at DESC
LIMIT $4 OFFSET $5;

-- name: CountFHIRResourcesByType :one
SELECT COUNT(*)
FROM records.fhir_resources
WHERE patient_id = $1
  AND resource_type = $2;

-- name: DeleteFHIRResourcesByPatientID :exec
DELETE FROM records.fhir_resources
WHERE patient_id = $1;

-- name: UpsertFHIRPatientMap :exec
INSERT INTO records.fhir_patient_map (fhir_patient_id, source_system, internal_patient_id)
VALUES ($1, $2, $3)
ON CONFLICT (fhir_patient_id, source_system)
DO UPDATE SET internal_patient_id = EXCLUDED.internal_patient_id;

-- name: GetInternalPatientIDByFHIR :one
SELECT internal_patient_id
FROM records.fhir_patient_map
WHERE fhir_patient_id = $1
  AND source_system = $2;

-- name: ListFHIRResourcesForPatient :many
SELECT
  id,
  resource_type,
  resource_id,
  resource_json::text AS resource_json,
  display_text,
  clinical_status
FROM records.fhir_resources
WHERE patient_id = $1
ORDER BY indexed_at DESC
LIMIT $2;

-- name: GetFHIRResourceByID :one
SELECT id, resource_type, resource_id, resource_json::text AS resource_json
FROM records.fhir_resources
WHERE patient_id = $1
  AND resource_type = $2
  AND resource_id = $3;
