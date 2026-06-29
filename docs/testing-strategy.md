# Testing Strategy

This document tracks the intended test pyramid for Zorba Health.

## Fast CI subset

These should stay runnable on every pull request:

- focused Go unit tests under changed services
- generated-code checks (`make generate-proto`, `make sqlc`, clean diff)
- notification / translation / health-records unit and contract tests
- demo smoke checks when deterministic seed data is available

## Unit tests

Priority coverage areas:

- auth logic and refresh token rotation
- patient registration and consent logic
- FHIR parsing and normalization
- RAG retrieval and citation enforcement
- MCP permission checks
- safety classifier and escalation logic
- audit logging helpers

## Integration tests

Target flows:

- API gateway → gRPC backend
- patient registration → notification
- voice agent → MCP server
- health records → pgvector / Postgres
- RabbitMQ publisher / consumer flows

## E2E tests

Target scenarios:

- patient registration
- grounded record retrieval
- translation with low-confidence advisory
- emergency escalation path
- notification delivery

## Tooling

- `go test` for unit and service-level integration tests
- `scripts/evaluation/demo-smoke.mjs` for API smoke coverage
- `k6` under `scripts/evaluation/load/` for latency and concurrency
- future: Playwright for web, Newman for Postman collections, fuller containerized integration suites

## Nightly / extended suite

Longer-running checks should move to a scheduled workflow once the deterministic local seed path stabilizes.
