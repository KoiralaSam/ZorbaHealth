# Health Records Service

Go gRPC service that owns patient-linked FHIR storage, pgvector chunk indexes, and the RAG pipeline used by MCP tools and the patient portal.

## Boundaries

| Layer | Responsibility | Path |
| --- | --- | --- |
| Service shell | gRPC auth, wiring, tracing | `cmd/`, `internal/adapters/primary/` |
| Core domain | Use-case orchestration | `internal/core/` |
| FHIR adapter | Bundle validation, normalization, ingestion | `internal/fhir/` |
| RAG module | Consent-aware retrieve → rerank → summarize | `internal/rag/` |
| Persistence | SQL / pgvector | `internal/adapters/secondary/repositories/postgres/` |

**Do not** import `internal/fhir` or `internal/rag` from other microservices. Call this service over gRPC (`HealthRecordService`).

## RAG pipeline (12 steps)

1. Receive patient-scoped query (gRPC).
2. Verify `HEALTH_RECORD_ACCESS` consent via `audit-service`.
3. Load FHIR linkage metadata from `records.fhir_resources`.
4. Chunk ingested narrative text at index time (`internal/rag/chunker`).
5. Embed query via configured embedder (OpenAI today).
6. Search `records.record_chunks` (pgvector).
7. ANN retrieval with patient filter.
8. Optional metadata filter (`fhir_resource_type`, access level).
9. Lightweight rerank (keyword + resource-type boosts).
10. Summarize **only retrieved chunks** when answering questions.
11. Return citations (`chunk_id`, `record_id`, `fhir_resource_type`).
12. Append `HEALTH_RECORD_SEARCHED` / `HEALTH_RECORD_SUMMARIZED` audit events.

## Database

Migration `000008_records_rag_fhir` creates schema `records` with:

- `records.fhir_resources`
- `records.fhir_patient_map`
- `records.record_chunks` (`vector(1536)` default)

Index tuning notes: [`docs/pgvector-indexing.md`](../../docs/pgvector-indexing.md).

## Local FHIR samples

- Bundle: [`examples/sample-fhir-data/demo-patient-bundle.json`](../../examples/sample-fhir-data/demo-patient-bundle.json)
- Seed CLI: [`scripts/seed-fhir-data/README.md`](../../scripts/seed-fhir-data/README.md)

## gRPC highlights

- `IngestFHIRBundle` — validates bundle, upserts FHIR rows, indexes chunks.
- `SearchRecords` — consent-aware vector search with citation metadata.
- `AnswerPatientQuestion` — grounded answer + citations.

Internal callers must send `x-internal-token` (`INTERNAL_SERVICE_SECRET`).
