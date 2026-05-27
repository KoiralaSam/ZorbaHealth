DROP MATERIALIZED VIEW IF EXISTS mv_hospital_top_conditions;

CREATE TABLE IF NOT EXISTS fhir_resources (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id    UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL,
    resource_id   TEXT NOT NULL,
    source_system TEXT,
    resource_json JSONB NOT NULL,
    indexed_at    TIMESTAMPTZ DEFAULT now(),
    UNIQUE (patient_id, resource_type, resource_id)
);

CREATE TABLE IF NOT EXISTS record_chunks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    source_file TEXT NOT NULL,
    chunk_index INT NOT NULL,
    chunk_text  TEXT NOT NULL,
    embedding   vector(1536),
    created_at  TIMESTAMPTZ DEFAULT now()
);

INSERT INTO fhir_resources (
    patient_id, resource_type, resource_id, source_system, resource_json, indexed_at
)
SELECT patient_id, resource_type, resource_id, source_system, resource_json, indexed_at
FROM records.fhir_resources;

INSERT INTO record_chunks (
    id, patient_id, source_file, chunk_index, chunk_text, embedding, created_at
)
SELECT
    chunk_id, patient_id, source_file, chunk_index, chunk_text, embedding, created_at
FROM records.record_chunks;

DROP TABLE IF EXISTS records.record_chunks;
DROP TABLE IF EXISTS records.fhir_resources;
DROP TABLE IF EXISTS records.fhir_patient_map;
DROP SCHEMA IF EXISTS records;

DO $$
BEGIN
    IF to_regclass('public.patient_hospital_consents') IS NOT NULL
       AND to_regclass('public.fhir_resources') IS NOT NULL
       AND to_regclass('public.mv_hospital_top_conditions') IS NULL THEN
        EXECUTE $sql$
        CREATE MATERIALIZED VIEW mv_hospital_top_conditions AS
        SELECT
            c.hospital_id,
            f.resource_json->>'resourceType' AS resource_type,
            f.resource_json->'code'->>'text' AS condition_name,
            COUNT(*) AS patient_count
        FROM patient_hospital_consents c
        JOIN fhir_resources f ON f.patient_id = c.patient_id
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
