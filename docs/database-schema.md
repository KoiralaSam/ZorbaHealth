# Database Schema

Zorba Health organizes data into schema-aligned ownership boundaries:

## Schemas

- `auth` (target) for auth/session/refresh-token records
- `patient` (target) for patient lifecycle and hospital relationship records
- `records` for FHIR resources, patient maps, and RAG chunks
- `audit` for append-only compliance evidence and consent history
- `analytics` for aggregated or compatibility analytics projections

## Current implementation notes

- Some legacy tables still remain in `public` while the repository migrates toward explicit schema ownership.
- `records.*` and `audit.*` now exist and should be preferred for new work.
- `analytics-service` should query `analytics.*` views/materialized views rather than joining raw FHIR resources directly.

## Retention expectations

- Pending registration state lives in Redis and should expire automatically.
- Live location session data is ephemeral and keyed by session TTL.
- Raw transcripts are off by default.
- Audit data follows explicit retention policy, not ad hoc deletion.
- Analytics projections can have a different retention window than audit history.

## Index priorities

- `patient_id`
- `created_at`
- `event_type`
- `correlation_id`
- pgvector indexes on `records.record_chunks.embedding`

## Contributor rule

New tables should be assigned to an explicit schema rather than added to `public` unless the change is part of a documented compatibility window.
