-- Enable the pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Chunks of patient health documents (PDFs, images, FHIR)
CREATE TABLE IF NOT EXISTS record_chunks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    source_file TEXT NOT NULL,
    chunk_index INT NOT NULL,
    chunk_text  TEXT NOT NULL,
    embedding   vector(1536),
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- Index for fast cosine similarity search
CREATE INDEX IF NOT EXISTS record_chunks_embedding_idx
    ON record_chunks USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- Conversation memory — every STT/agent turn
CREATE TABLE IF NOT EXISTS conversation_turns (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID NOT NULL,
    session_id  TEXT NOT NULL,
    role        TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    content     TEXT NOT NULL,
    embedding   vector(1536),
    created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS conversation_turns_patient_idx
    ON conversation_turns (patient_id, created_at DESC);

-- Audit log — append-only, never deleted
CREATE TABLE IF NOT EXISTS mcp_audit_log (
    id          BIGSERIAL PRIMARY KEY,
    timestamp   TIMESTAMPTZ DEFAULT now(),
    session_id  TEXT NOT NULL,
    patient_id  TEXT NOT NULL,
    tool        TEXT NOT NULL,
    outcome     TEXT NOT NULL,
    error_msg   TEXT
);