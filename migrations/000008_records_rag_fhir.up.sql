-- Phase 3: records schema for FHIR storage and pgvector RAG chunks.

CREATE SCHEMA IF NOT EXISTS records;

CREATE TABLE IF NOT EXISTS records.fhir_patient_map (
    fhir_patient_id     TEXT NOT NULL,
    source_system       TEXT NOT NULL DEFAULT 'unknown',
    internal_patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (fhir_patient_id, source_system)
);

CREATE INDEX IF NOT EXISTS fhir_patient_map_internal_idx
    ON records.fhir_patient_map (internal_patient_id);

CREATE TABLE IF NOT EXISTS records.fhir_resources (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id       UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    resource_type    TEXT NOT NULL,
    resource_id      TEXT NOT NULL,
    source_system    TEXT,
    resource_json    JSONB NOT NULL,
    display_text     TEXT,
    clinical_status  TEXT,
    effective_date   TIMESTAMPTZ,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (patient_id, resource_type, resource_id)
);

CREATE INDEX IF NOT EXISTS fhir_resources_patient_type_idx
    ON records.fhir_resources (patient_id, resource_type);
CREATE INDEX IF NOT EXISTS fhir_resources_json_idx
    ON records.fhir_resources USING GIN (resource_json);

INSERT INTO records.fhir_resources (
    id, patient_id, resource_type, resource_id, source_system, resource_json, indexed_at
)
SELECT id, patient_id, resource_type, resource_id, source_system, resource_json, indexed_at
FROM fhir_resources
ON CONFLICT (patient_id, resource_type, resource_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS records.record_chunks (
    chunk_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id         UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    record_id          UUID REFERENCES records.fhir_resources(id) ON DELETE SET NULL,
    fhir_resource_type TEXT,
    source_system      TEXT,
    source_file        TEXT NOT NULL DEFAULT '',
    chunk_index        INT NOT NULL DEFAULT 0,
    chunk_text         TEXT NOT NULL,
    chunk_hash         TEXT NOT NULL,
    access_level       TEXT NOT NULL DEFAULT 'patient' CHECK (access_level IN ('patient', 'provider', 'system')),
    embedding_model    TEXT NOT NULL DEFAULT 'text-embedding-3-small',
    embedding          vector(1536),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS record_chunks_patient_created_idx
    ON records.record_chunks (patient_id, created_at DESC);
CREATE INDEX IF NOT EXISTS record_chunks_patient_type_idx
    ON records.record_chunks (patient_id, fhir_resource_type);
CREATE INDEX IF NOT EXISTS record_chunks_hash_idx
    ON records.record_chunks (patient_id, chunk_hash);

-- Default ANN index (lists tuned for small dev datasets; see docs/pgvector-indexing.md).
CREATE INDEX IF NOT EXISTS record_chunks_embedding_idx
    ON records.record_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

INSERT INTO records.record_chunks (
    chunk_id, patient_id, fhir_resource_type, source_system, source_file, chunk_index,
    chunk_text, chunk_hash, access_level, embedding_model, embedding, created_at, updated_at
)
SELECT
    id,
    patient_id,
    NULLIF(split_part(source_file, ':', 1), ''),
    NULLIF(split_part(source_file, ':', 2), ''),
    source_file,
    chunk_index,
    chunk_text,
    encode(sha256(chunk_text::bytea), 'hex'),
    'patient',
    'text-embedding-3-small',
    embedding,
    created_at,
    COALESCE(created_at, now())
FROM record_chunks
ON CONFLICT DO NOTHING;

-- mv_hospital_top_conditions (migration 000005) references public.fhir_resources.
DROP MATERIALIZED VIEW IF EXISTS mv_hospital_top_conditions;

DROP TABLE IF EXISTS record_chunks;
DROP TABLE IF EXISTS fhir_resources;

DO $$
BEGIN
    IF to_regclass('public.patient_hospital_consents') IS NOT NULL
       AND to_regclass('records.fhir_resources') IS NOT NULL
       AND to_regclass('public.mv_hospital_top_conditions') IS NULL THEN
        EXECUTE $sql$
        CREATE MATERIALIZED VIEW mv_hospital_top_conditions AS
        SELECT
            c.hospital_id,
            f.resource_json->>'resourceType' AS resource_type,
            f.resource_json->'code'->>'text' AS condition_name,
            COUNT(*) AS patient_count
        FROM patient_hospital_consents c
        JOIN records.fhir_resources f ON f.patient_id = c.patient_id
        WHERE c.revoked_at IS NULL
          AND f.resource_type = 'Condition'
          AND f.resource_json->'clinicalStatus'->'coding'->0->>'code' = 'active'
        GROUP BY
            c.hospital_id,
            f.resource_json->>'resourceType',
            f.resource_json->'code'->>'text'
        ORDER BY patient_count DESC
        WITH DATA;
        $sql$;

        EXECUTE $sql$
        CREATE UNIQUE INDEX idx_mv_hospital_top_conditions_hospital_condition
            ON mv_hospital_top_conditions (hospital_id, condition_name);
        $sql$;
    END IF;
END;
$$;
