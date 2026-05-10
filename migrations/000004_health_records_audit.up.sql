-- ── VECTOR EXTENSION ─────────────────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS vector;

-- ── HEALTH RECORDS ───────────────────────────────────────────────────────────
-- Owned exclusively by health-record-service. No other service queries this directly.

CREATE TABLE IF NOT EXISTS record_chunks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    source_file TEXT NOT NULL,
    chunk_index INT NOT NULL,
    chunk_text  TEXT NOT NULL,
    embedding   vector(1536),   -- text-embedding-3-small, 1536 dims
    created_at  TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS record_chunks_embedding_idx
    ON record_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Conversation memory — owned by health-record-service
CREATE TABLE IF NOT EXISTS conversation_turns (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID NOT NULL,
    session_id  TEXT NOT NULL,
    role        TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    content     TEXT NOT NULL,
    embedding   vector(1536),
    created_at  TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS conv_turns_patient_idx
    ON conversation_turns (patient_id, created_at DESC);

-- ── AUDIT LOG ─────────────────────────────────────────────────────────────────
-- Owned by mcp-server. Append-only — never updated or deleted.

CREATE TABLE IF NOT EXISTS mcp_audit_log (
    id          BIGSERIAL PRIMARY KEY,
    timestamp   TIMESTAMPTZ DEFAULT now(),
    session_id  TEXT NOT NULL,
    actor_type  TEXT NOT NULL CHECK (actor_type IN ('patient','staff','admin')),
    actor_id    TEXT NOT NULL,
    hospital_id TEXT,
    tool        TEXT NOT NULL,
    outcome     TEXT NOT NULL CHECK (outcome IN ('success','forbidden','consent-denied','error')),
    error_msg   TEXT
);

-- ── FHIR RESOURCES ────────────────────────────────────────────────────────────
-- Owned exclusively by health-record-service.
-- Stores structured FHIR R4 resources from hospital imports and manual uploads.
-- Coexists with record_chunks: FHIR resources give typed structured queries,
-- record_chunks give semantic vector search. Both are populated during ingestion.

CREATE TABLE IF NOT EXISTS fhir_resources (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id    UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL,   -- "Condition" | "MedicationStatement" | "Observation" | etc.
    resource_id   TEXT NOT NULL,   -- FHIR resource ID from the source system
    source_system TEXT,            -- "epic" | "cerner" | "manual-upload"
    resource_json JSONB NOT NULL,  -- full FHIR R4 resource JSON
    indexed_at    TIMESTAMPTZ DEFAULT now(),
    UNIQUE (patient_id, resource_type, resource_id)
);

CREATE INDEX IF NOT EXISTS fhir_resources_patient_type_idx
    ON fhir_resources (patient_id, resource_type);
CREATE INDEX IF NOT EXISTS fhir_resources_json_idx
    ON fhir_resources USING GIN (resource_json);
