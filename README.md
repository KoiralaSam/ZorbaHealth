# Zorba Health

Zorba Health is an open-source voice-based healthcare AI platform built around Go microservices, a Next.js web app, gRPC, RabbitMQ, PostgreSQL, Redis, and voice integrations such as LiveKit and FreePBX.

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

1. Clone the repository.
2. Copy the development Kubernetes secrets template:

```bash
cp deploy/kubernetes/development/secrets.example.yaml deploy/kubernetes/development/secrets.yaml
```

3. Replace every placeholder in `deploy/kubernetes/development/secrets.yaml`.
4. Start the local stack:

```bash
tilt up
```

5. Open:
   - Web app: `http://localhost:3000`
   - API gateway: `http://localhost:8081`
   - Tilt UI: `http://localhost:10350`

Detailed setup instructions live in:

- [`docs/local-setup.md`](docs/local-setup.md)
- [`docs/kubernetes-setup.md`](docs/kubernetes-setup.md)
- [`docs/deployment.md`](docs/deployment.md)

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — architecture overview, service boundaries, and future-work callouts
- [`docs/directory-map.md`](docs/directory-map.md) — how the requested open-source layout maps to the current tree
- [`docs/service-map.md`](docs/service-map.md) — service inventory, interfaces, dependencies, and maturity
- [`docs/security.md`](docs/security.md) — authentication, authorization, logging, and secrets posture
- [`docs/compliance.md`](docs/compliance.md) — audit, consent, analytics separation, and retention guidance
- [`docs/evaluation.md`](docs/evaluation.md) — evaluation artifacts and measurement roadmap
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

Voice and SIP infrastructure are not fully contained inside the local development stack. The expected production architecture includes:

- FreePBX for telephony
- LiveKit for real-time audio sessions
- LiveKit SIP for SIP-to-room bridging
- `voice-agent-service` for the active LiveKit Agents voice runtime and webhook handling

Those components are typically operated on separate infrastructure and integrated with the services in this repository.

## Community files

- [`CONTRIBUTING.md`](CONTRIBUTING.md)
- [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)
- [`SECURITY.md`](SECURITY.md)
- [`LICENSE`](LICENSE)

## Current local development expectations

- Go module root: repository root
- Local orchestration: Tilt + Kubernetes
- Secrets template: `deploy/kubernetes/development/secrets.example.yaml`
- Database migrations: `migrations/`
- Proto generation and SQLC generation: `Makefile`

If you are looking for the requested `/zorba-health/services`, `/proto`, or `/deploy` layout from the architecture suggestions, that layout is now the repository structure. [`docs/directory-map.md`](docs/directory-map.md) documents the detailed mapping and development subdirectories.
