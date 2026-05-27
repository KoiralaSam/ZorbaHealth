-- name: AppendAuditEvent :one
INSERT INTO audit.audit_events (
  event_id,
  event_type,
  actor_type,
  actor_id,
  patient_id,
  service_name,
  resource_type,
  resource_id,
  event_timestamp,
  request_id,
  correlation_id,
  ip_address,
  tool_name,
  model_name,
  provider_name,
  success_status,
  failure_reason,
  metadata_json
) VALUES (
  sqlc.arg(event_id),
  sqlc.arg(event_type),
  sqlc.arg(actor_type),
  sqlc.arg(actor_id),
  NULLIF(sqlc.arg(patient_id), ''),
  sqlc.arg(service_name),
  NULLIF(sqlc.arg(resource_type), ''),
  NULLIF(sqlc.arg(resource_id), ''),
  sqlc.arg(event_timestamp),
  NULLIF(sqlc.arg(request_id), ''),
  NULLIF(sqlc.arg(correlation_id), ''),
  NULLIF(sqlc.arg(ip_address), ''),
  NULLIF(sqlc.arg(tool_name), ''),
  NULLIF(sqlc.arg(model_name), ''),
  NULLIF(sqlc.arg(provider_name), ''),
  sqlc.arg(success_status),
  NULLIF(sqlc.arg(failure_reason), ''),
  sqlc.arg(metadata_json)::jsonb
)
RETURNING *;

-- name: QueryAuditEvents :many
SELECT *
FROM audit.audit_events
WHERE (sqlc.arg(event_type) = '' OR event_type = sqlc.arg(event_type))
  AND (sqlc.arg(actor_type) = '' OR actor_type = sqlc.arg(actor_type))
  AND (sqlc.arg(actor_id) = '' OR actor_id = sqlc.arg(actor_id))
  AND (sqlc.arg(patient_id) = '' OR patient_id = sqlc.arg(patient_id))
  AND (sqlc.arg(service_name) = '' OR service_name = sqlc.arg(service_name))
  AND (sqlc.arg(correlation_id) = '' OR correlation_id = sqlc.arg(correlation_id))
ORDER BY event_timestamp DESC
LIMIT sqlc.arg(result_limit);

-- name: ListPatientPortalAuditEvents :many
SELECT *
FROM audit.audit_events
WHERE patient_id = sqlc.arg(patient_id)
ORDER BY event_timestamp DESC
LIMIT sqlc.arg(result_limit);

-- name: ListHospitalPortalIncidents :many
SELECT *
FROM audit.audit_events
WHERE event_type = sqlc.arg(event_type)
ORDER BY event_timestamp DESC
LIMIT sqlc.arg(result_limit);

-- name: GetLatestMatchingConsent :one
SELECT *
FROM audit.consents
WHERE patient_id = sqlc.arg(patient_id)
  AND consent_type = sqlc.arg(consent_type)
  AND (sqlc.arg(scope_value) = '' OR scope = sqlc.arg(scope_value) OR scope = '')
ORDER BY granted_at DESC
LIMIT 1;

-- name: GrantConsent :one
INSERT INTO audit.consents (
  id,
  patient_id,
  consent_type,
  granted_by,
  granted_at,
  scope,
  expiration_time,
  source,
  metadata_json
) VALUES (
  sqlc.arg(id),
  sqlc.arg(patient_id),
  sqlc.arg(consent_type),
  sqlc.arg(granted_by),
  sqlc.arg(granted_at),
  sqlc.arg(scope),
  sqlc.narg(expiration_time),
  sqlc.arg(source),
  sqlc.arg(metadata_json)::jsonb
)
RETURNING *;

-- name: RevokeConsent :one
UPDATE audit.consents AS c
SET revoked_at = now(),
    source = CASE WHEN sqlc.arg(source) <> '' THEN sqlc.arg(source) ELSE c.source END,
    metadata_json = sqlc.arg(metadata_json)::jsonb
WHERE id = (
  SELECT c2.id
  FROM audit.consents AS c2
  WHERE c2.patient_id = sqlc.arg(patient_id)
    AND c2.consent_type = sqlc.arg(consent_type)
    AND (sqlc.arg(scope_value) = '' OR c2.scope = sqlc.arg(scope_value) OR c2.scope = '')
    AND c2.revoked_at IS NULL
  ORDER BY c2.granted_at DESC
  LIMIT 1
)
RETURNING *;

-- name: ListConsents :many
SELECT *
FROM audit.consents
WHERE (sqlc.arg(patient_id) = '' OR patient_id = sqlc.arg(patient_id))
  AND (sqlc.arg(consent_type) = '' OR consent_type = sqlc.arg(consent_type))
  AND (sqlc.arg(include_revoked)::bool OR revoked_at IS NULL)
ORDER BY granted_at DESC
LIMIT sqlc.arg(result_limit);

-- name: ListPatientPortalConsents :many
SELECT *
FROM audit.consents
WHERE patient_id = sqlc.arg(patient_id)
  AND (sqlc.arg(include_revoked)::bool OR revoked_at IS NULL)
ORDER BY granted_at DESC
LIMIT sqlc.arg(result_limit);

-- name: CheckHospitalConsentAccess :one
SELECT EXISTS (
  SELECT 1
  FROM patient_hospital_consents
  WHERE patient_id = sqlc.arg(patient_id)::uuid
    AND hospital_id = sqlc.arg(hospital_id)::uuid
    AND revoked_at IS NULL
);
