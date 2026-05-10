# Directory Map

Zorba Health now uses the repository root as the canonical workspace root. The Phase 1 repository cleanup promoted the former `src/*` tree into the top-level layout requested by the architecture plan.

## Canonical workspace

- `go.mod` — Go module root
- `Tiltfile` — local development entrypoint
- `Makefile` — protobuf generation and migration helpers

## Repository layout

```text
zorba-health/
  services/
  web/
  proto/
  shared/
  deploy/
    docker/
      development/
      production/
    kubernetes/
      development/
      production/
    helm/
  docs/
    architecture/
  examples/
  scripts/
  migrations/
  tools/
```

## Directory responsibilities

- `services/` — Go microservices
- `web/` — Next.js frontend
- `proto/` — source protobuf definitions
- `shared/` — shared Go packages and generated protobuf code
- `deploy/docker/` — development and production Dockerfiles
- `deploy/kubernetes/` — development and production Kubernetes manifests
- `deploy/helm/` — reserved for future Helm charts
- `docs/` — architecture, setup, security, compliance, and evaluation docs
- `examples/` — sample environment variables, request examples, and sample-data placeholders
- `scripts/` — migration, seed-data, and evaluation script entrypoints
- `migrations/` — SQL migrations
- `tools/` — repository helpers and utility programs

## Compatibility notes

The old `src/` layout has been retired in favor of the target top-level structure. Build and development entrypoints were moved with it:

- `src/go.mod` -> `go.mod`
- `src/Tiltfile` -> `Tiltfile`
- `src/Makefile` -> `Makefile`
- `src/services` -> `services`
- `src/web` -> `web`
- `src/proto` -> `proto`
- `src/shared` -> `shared`
- `src/migrations` -> `migrations`
- `src/tools` -> `tools`
- `src/infra/development/docker` -> `deploy/docker/development`
- `src/infra/production/docker` -> `deploy/docker/production`
- `src/infra/development/k8s` -> `deploy/kubernetes/development`
- `src/infra/production/k8s` -> `deploy/kubernetes/production`
- `src/docs/architecture` -> `docs/architecture`

## Contributor guidance

Run repository-level development commands from the repository root unless a subproject explicitly says otherwise. The main exception is the frontend package manager workflow, which runs from `web/`.
