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

The long-term architecture expects provider interfaces for:

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

The current codebase still has direct provider coupling in several services, so contributions in this area should:

1. Introduce a clear outbound interface in the relevant service or shared package.
2. Move provider-specific code into an adapter package.
3. Keep provider-specific configuration out of core domain logic.
4. Update security and setup docs with any new environment variables.

## Add a new MCP tool

1. Implement the tool under `services/mcp-server/tools`.
2. Register it in `services/mcp-server/cmd/mcp-server/main.go`.
3. Document its purpose, expected permissions, and service dependencies.
4. Prefer returning minimal, safe context rather than raw records.
5. Update `docs/service-map.md` or `docs/architecture.md` if the new tool changes service boundaries.

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
