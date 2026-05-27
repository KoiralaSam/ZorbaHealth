# Contributing to Zorba Health

Thanks for contributing to Zorba Health.

## Before you open a pull request

1. Read `README.md`
2. Read the docs relevant to your change under `docs/`
3. Make sure your change fits the current top-level repository layout

## Development workflow

The Go module root is the repository root, so most contributor commands run there:

```bash
make generate-proto
make sqlc
make migrate-up
go test ./...
```

For frontend changes:

```bash
cd web
npm install
npm test
```

## Contribution expectations

- keep secrets out of commits
- update docs when architecture, setup, or security expectations change
- keep service boundaries explicit
- prefer small, reviewable pull requests
- document future work instead of silently leaving architectural gaps

## Migrations

- place SQL migrations under `migrations`
- keep up/down migration pairs together
- document schema changes if they affect contributors or deployment workflows

## Protobuf contracts

- edit source `.proto` files under `proto`
- regenerate code from the repository root

## Pull request checklist

- tests pass locally for the changed area
- generated code is checked in when required
- docs are updated when behavior or setup changed
- no credentials or sensitive data were added

## Scope notes

The project is moving toward a more modular, provider-agnostic, compliance-aware architecture. If your change touches those areas, explain how it affects:

- service boundaries
- data handling
- external provider coupling
- audit and consent implications

### Architecture boundary lint (health records)

When editing `services/health-records-service`:

- keep FHIR parsing in `internal/fhir`
- keep retrieval/rerank/summarize orchestration in `internal/rag`
- keep SQL in `internal/adapters/secondary/repositories/postgres`
- do not import `internal/rag` or `internal/fhir` from other services

See [`services/health-records-service/README.md`](services/health-records-service/README.md) and the **Health records service vs RAG module** section in [`docs/architecture.md`](docs/architecture.md).
