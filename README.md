# Zorba Health

Zorba Health is an open-source voice-based healthcare AI platform built around Go microservices, a Next.js web app, gRPC, RabbitMQ, PostgreSQL, Redis, and voice integrations such as LiveKit and LiveKit SIP.

## What is in this repository

The repository is now organized around the top-level service and deployment layout:

- `services/` contains the Go microservices
- `web/` contains the Next.js frontend
- `proto/` contains source protobuf definitions
- `shared/` contains shared Go packages and generated protobuf code
- `migrations/` contains SQL migrations
- `deploy/` contains Dockerfiles and Kubernetes manifests

This repo also includes contributor-facing assets under `docs/`, `examples/`, and `scripts/`.

## Quick start

### Docker Compose (recommended)

```bash
docker compose \
  -f deploy/docker/docker-compose.yml \
  -f deploy/docker/docker-compose.override.local.yml \
  up --build
```

Then migrate:

```bash
export DATABASE_URL='postgres://healthai:healthai@localhost:5432/healthai?sslmode=disable'
make migrate-up
```

Open:
- Web app: `http://localhost:3000`
- API gateway: `http://localhost:8081`

### GitHub Codespaces

Create a Codespace from this repo (8-core / 16GB+ recommended; 64GB storage preferred for kind+Tilt).

**Compose (lighter):**

```bash
./scripts/codespaces/prepare-env.sh
docker compose \
  -f deploy/docker/docker-compose.yml \
  -f deploy/docker/docker-compose.override.codespaces.yml \
  up --build
```

**kind + Tilt (Kubernetes in Docker):**

```bash
./scripts/codespaces/kind-up.sh
# fill deploy/kubernetes/development/secrets.yaml
./deploy/tilt/preflight.sh && tilt up
```

Set Ports panel visibility to **Public** for `3000` / `8081` / `8091` when using `*.app.github.dev` URLs. Full steps: [`docs/local-setup.md`](docs/local-setup.md#option-b-github-codespaces).

### Optional: Tilt + local Kubernetes

```bash
cp deploy/kubernetes/development/secrets.example.yaml deploy/kubernetes/development/secrets.yaml
# fill placeholders, then:
./deploy/tilt/preflight.sh
tilt up
```

Detailed setup: [`docs/local-setup.md`](docs/local-setup.md), [`docs/kubernetes-setup.md`](docs/kubernetes-setup.md), [`docs/deployment.md`](docs/deployment.md).

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — architecture overview, service boundaries, and future-work callouts
- [`docs/directory-map.md`](docs/directory-map.md) — how the requested open-source layout maps to the current tree
- [`docs/service-map.md`](docs/service-map.md) — service inventory, interfaces, dependencies, and maturity
- [`docs/security.md`](docs/security.md) — authentication, authorization, logging, and secrets posture
- [`docs/compliance.md`](docs/compliance.md) — audit, consent, analytics separation, and retention guidance
- [`docs/evaluation.md`](docs/evaluation.md) — evaluation artifacts and measurement roadmap
- [`docs/database-schema.md`](docs/database-schema.md) — schema ownership, retention, and indexing notes
- [`docs/testing-strategy.md`](docs/testing-strategy.md) — unit, integration, smoke, and longer-running test guidance
- [`docs/observability.md`](docs/observability.md) — OpenTelemetry, metrics, dashboards, and trace expectations
- [`docs/open-source-extension-guide.md`](docs/open-source-extension-guide.md) — how to add providers, services, tools, and migrations
- [`docs/q1-journal-evaluation-plan.md`](docs/q1-journal-evaluation-plan.md) — research framing and target metrics

Existing architecture flow notes from the implementation are kept under:

- [`docs/architecture/rabbitmq-flow-v1.md`](docs/architecture/rabbitmq-flow-v1.md)
- [`docs/architecture/voice-call-flow-v1.md`](docs/architecture/voice-call-flow-v1.md)

## Repository status

Current primary services:

- API gateway
- Auth service
- Patient service
- Health records service
- MCP server
- Voice agent service
- Notification service
- Location service
- Translation service
- Analytics service

Some parts of the architecture described in the long-term roadmap are still future work, including a dedicated audit service, consent enforcement service boundaries, Helm packaging, richer provider abstraction, and a formal evaluation harness. Those gaps are documented explicitly rather than hidden.

## Voice and telephony note

LiveKit (rooms) and LiveKit SIP (PSTN bridge) run as a **host Docker sidecar** for local development — see [`deploy/docker/livekit/`](deploy/docker/livekit/). FreePBX is not required; a VoIP.ms DID can route voice straight to LiveKit SIP.

- LiveKit for real-time audio sessions
- LiveKit SIP for SIP-to-room bridging and dispatch rules
- `voice-agent-service` for the LiveKit Agents runtime and webhook handling

Production may still run LiveKit on separate infrastructure; the Compose sidecar is the supported local path alongside Tilt/minikube.

## Community files

- [`CONTRIBUTING.md`](CONTRIBUTING.md)
- [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)
- [`SECURITY.md`](SECURITY.md)
- [`LICENSE`](LICENSE)

## Current local development expectations

- Go module root: repository root
- Primary local orchestration: Docker Compose (`deploy/docker/`)
- Optional local Kubernetes: Tilt (`Tiltfile`) + `deploy/kubernetes/development/`
- Secrets template: `deploy/kubernetes/development/secrets.example.yaml`
- Database migrations: `migrations/`
- Proto generation and SQLC generation: `Makefile`

If you are looking for the requested `/zorba-health/services`, `/proto`, or `/deploy` layout from the architecture suggestions, that layout is now the repository structure. [`docs/directory-map.md`](docs/directory-map.md) documents the detailed mapping and development subdirectories.

## Research Alignment

The current implementation and repo artifacts are organized around five evidence-backed claims:

1. An open-source voice healthcare architecture documented in `docs/architecture.md` and `docs/service-map.md`
2. A controlled MCP gateway backed by auth, consent, and audit boundaries in `docs/compliance.md`
3. A FHIR-compatible RAG retrieval path demonstrated by the sample bundle and evaluation smoke scripts
4. A safety / consent / audit framework documented in `docs/security.md` and implemented across `audit-service`, `mcp-server`, and `voice-agent-service`
5. A reproducible deployment and evaluation path through `deploy/docker/docker-compose.yml`, `docs/local-setup.md`, and `scripts/evaluation/run_all.sh`
