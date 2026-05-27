-- Vector search and ingestion for records.record_chunks

-- name: CreateRecordChunk :one
INSERT INTO records.record_chunks (
  patient_id,
  record_id,
  fhir_resource_type,
  source_system,
  source_file,
  chunk_index,
  chunk_text,
  chunk_hash,
  access_level,
  embedding_model,
  embedding
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::vector
)
RETURNING *;

-- name: ListRecordChunksByPatientID :many
SELECT *
FROM records.record_chunks
WHERE patient_id = $1
ORDER BY created_at DESC, chunk_index ASC
LIMIT $2 OFFSET $3;

-- name: DeleteRecordChunksByPatientID :exec
DELETE FROM records.record_chunks
WHERE patient_id = $1;

-- name: SearchRecordChunksByEmbedding :many
SELECT
  chunk_id,
  chunk_text,
  source_file,
  record_id,
  fhir_resource_type,
  (1 - (embedding <=> $2::vector))::float4 AS score
FROM records.record_chunks
WHERE patient_id = $1
  AND ($4 = '' OR fhir_resource_type = $4)
  AND access_level IN ('patient', 'provider')
ORDER BY embedding <=> $2::vector
LIMIT $3;

-- name: HospitalSearchRecordChunksByEmbedding :many
SELECT
  rc.chunk_id,
  rc.chunk_text,
  rc.source_file,
  rc.record_id,
  rc.fhir_resource_type,
  (1 - (rc.embedding <=> $3::vector))::float4 AS score
FROM records.record_chunks rc
WHERE rc.patient_id = $1
  AND rc.access_level IN ('patient', 'provider')
  AND EXISTS (
    SELECT 1
    FROM patient_hospital_consents phc
    WHERE phc.patient_id = $1
      AND phc.hospital_id = $2
      AND phc.revoked_at IS NULL
  )
ORDER BY rc.embedding <=> $3::vector
LIMIT $4;

-- name: FetchChunksForSummary :many
SELECT
  chunk_text
FROM records.record_chunks
WHERE patient_id = $1
  AND (
    $2 = '' OR
    $2 = 'full' OR
    lower(chunk_text) LIKE '%' || lower($2) || '%'
  )
ORDER BY chunk_index
LIMIT $3;

-- name: SearchRecordChunksCandidates :many
SELECT
  chunk_id,
  chunk_text,
  source_file,
  record_id,
  fhir_resource_type,
  (1 - (embedding <=> $2::vector))::float4 AS score
FROM records.record_chunks
WHERE patient_id = $1
  AND access_level IN ('patient', 'provider')
ORDER BY embedding <=> $2::vector
LIMIT $3;
