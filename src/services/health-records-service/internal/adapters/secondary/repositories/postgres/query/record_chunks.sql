-- Vector search and ingestion for record_chunks

-- name: CreateRecordChunk :one
INSERT INTO record_chunks (
  patient_id,
  source_file,
  chunk_index,
  chunk_text,
  embedding
) VALUES (
  $1, $2, $3, $4, $5::vector
)
RETURNING *;

-- name: ListRecordChunksByPatientID :many
SELECT *
FROM record_chunks
WHERE patient_id = $1
ORDER BY created_at DESC, chunk_index ASC
LIMIT $2 OFFSET $3;

-- name: DeleteRecordChunksByPatientID :exec
DELETE FROM record_chunks
WHERE patient_id = $1;

-- name: SearchRecordChunksByEmbedding :many
SELECT
  chunk_text,
  source_file,
  (1 - (embedding <=> $2::vector))::float4 AS score
FROM record_chunks
WHERE patient_id = $1
ORDER BY embedding <=> $2::vector
LIMIT $3;

-- name: HospitalSearchRecordChunksByEmbedding :many
SELECT
  rc.chunk_text,
  rc.source_file,
  (1 - (rc.embedding <=> $3::vector))::float4 AS score
FROM record_chunks rc
WHERE rc.patient_id = $1
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
FROM record_chunks
WHERE patient_id = $1
  AND (
    $2 = '' OR
    $2 = 'full' OR
    lower(chunk_text) LIKE '%' || lower($2) || '%'
  )
ORDER BY chunk_index
LIMIT $3;


