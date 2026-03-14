-- Drop tables (reverse order of creation)
DROP TABLE IF EXISTS mcp_audit_log;
DROP TABLE IF EXISTS conversation_turns;
DROP TABLE IF EXISTS record_chunks;

-- Drop the pgvector extension
DROP EXTENSION IF EXISTS vector;
