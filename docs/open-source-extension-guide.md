# Open-Source Extension Guide

This guide explains how contributors should extend the current Zorba Health repository structure.

## Add a new Go service

1. Create the service under `services/<name>-service`.
2. Follow the existing service structure:
   - `cmd/<name>-service/main.go`
   - `internal/adapters/primary`
   - `internal/adapters/secondary`
   - `internal/core`
   - `config`
3. Add protobuf contracts under `proto` if the service needs gRPC APIs.
4. Generate code from the repository root using `make generate-proto`.
5. Add migrations under `migrations` if the service needs database changes.
6. Add Docker and Kubernetes assets under `deploy/`.
7. Update `docs/service-map.md` and `docs/architecture.md`.

There is also an existing helper at `tools/create_service.go` that reflects the current service layout.

## Add a new provider integration

The shared provider port package now lives at `shared/ports/providers` and defines provider-facing contracts for:

- LLM
- embeddings
- speech-to-text
- text-to-speech
- email
- SMS
- telephony
- translation
- vector search
- FHIR

The current codebase still has some concrete adapters, but the expected contribution path is:

1. Implement the shared provider contract or add a new one under `shared/ports/providers`.
2. Keep provider-specific code in an adapter package (`internal/adapters/secondary/...`).
3. Select the adapter in wiring via config (`LLM_PROVIDER`, `EMBEDDING_PROVIDER`, `EMAIL_PROVIDER`, `SMS_PROVIDER`, etc.).
4. Keep provider-specific configuration out of core domain logic.
5. Update security and setup docs with any new environment variables.

## Add a new MCP tool

1. Implement the tool under `services/mcp-server/tools`.
2. Register it in `services/mcp-server/cmd/mcp-server/main.go`.
3. Document its purpose, expected permissions, and service dependencies.
4. Prefer returning minimal, safe context rather than raw records.
5. Update `docs/service-map.md` or `docs/architecture.md` if the new tool changes service boundaries.

## Add a patient notification workflow

Use this checklist when adding email/SMS tied to a product workflow (not generic send):

1. **Voice / HTTP entry:** `@function_tool` in `services/voice-agent-service` and/or routes in `services/api-gateway` with structured JSON only (no message body from the LLM).
2. **gRPC + optional MCP:** define RPCs under `proto/`, implement orchestration in the owning service (`internal/core/services`), expose via gRPC primary adapter; add a thin MCP tool in `services/mcp-server/tools` for voice if needed.
3. **Orchestration:** consent and validation in core; publish a typed event via `shared/events` and `shared/contracts` routing keys (prefer patient-service as publisher).
4. **Notification-service:** RabbitMQ consumer in `internal/adapters/primary/events/rabbitmq`, templated method on `internal/core/ports/inbound/notification_service.go`, provider adapters for email/SMS, `NOTIFICATION_SENT` audit via audit-service.
5. **Observability:** propagate `correlation_id`, OTel spans, audit events (`MEETING_SCHEDULED`, denials, per-channel `NOTIFICATION_SENT`).

Reference implementation: health-staff meeting scheduling (`SchedulingService`, `schedule_health_staff_meeting`, `patient.event.meeting_scheduled`).

## Add a database migration

1. Create a numbered migration pair under `migrations`.
2. Run migrations from the repository root.
3. Regenerate SQLC output if repository queries change.
4. Document schema-impacting changes in contributor-facing docs when appropriate.

## Add or change protobuf contracts

1. Edit `.proto` files under `proto`.
2. From the repository root, run:

```bash
make generate-proto
```

3. Verify generated Go code in `shared/proto`.

## Documentation expectations

When adding meaningful architecture or platform changes, update the relevant documents:

- `docs/architecture.md`
- `docs/service-map.md`
- `docs/local-setup.md`
- `docs/security.md`
- `docs/compliance.md`
- `docs/evaluation.md`
