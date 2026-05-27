# pgvector indexing

Health record semantic search stores embeddings in `records.record_chunks.embedding` as `vector(1536)` (OpenAI `text-embedding-3-small`).

## Development default

Migration `000008_records_rag_fhir` creates an **IVFFlat** index with `lists = 100`. This is appropriate for small local datasets. After bulk ingestion in shared environments, consider `REINDEX` or rebuilding with a higher `lists` value (~sqrt(row count)).

## Production tuning

- **HNSW** often gives better recall/latency trade-offs on PostgreSQL 16+ with recent pgvector releases. Example:

```sql
CREATE INDEX record_chunks_embedding_hnsw_idx
  ON records.record_chunks USING hnsw (embedding vector_cosine_ops);
```

- Run `ANALYZE records.record_chunks` after large imports.
- Keep `patient_id` filters in queries so the planner can combine btree + vector indexes.

## Dimension changes

If you change embedding models, alter the column dimension and re-embed all chunks. Document the active model in `embedding_model` on each row.
