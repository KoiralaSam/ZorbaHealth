# Local Setup

This project uses the repository root as the working directory for Go builds, migrations, Tilt, and protobuf generation.

## Prerequisites

- Docker Desktop or another local Docker runtime
- A local Kubernetes cluster such as Docker Desktop Kubernetes or Minikube
- `kubectl`
- Tilt
- Go
- `migrate`
- `protoc`
- `sqlc`

## Clone the repository

```bash
git clone https://github.com/KoiralaSam/ZorbaHealth.git
cd ZorbaHealth
```

## Prepare secrets

Copy the Kubernetes secrets template:

```bash
cp deploy/kubernetes/development/secrets.example.yaml deploy/kubernetes/development/secrets.yaml
```

Fill every placeholder before starting Tilt.

For a contributor-friendly view of environment variables, also see:

- `examples/sample-env/.env.example`

## Start the local stack

### Option A: Docker Compose

For a single-command OSS setup path (prefer the local override so services use Compose-friendly defaults):

```bash
docker compose \
  -f deploy/docker/docker-compose.yml \
  -f deploy/docker/docker-compose.override.local.yml \
  up --build
```

Without the override, the base file still works but loads placeholder values from `examples/sample-env/.env.example`.

This starts Postgres (pgvector), Redis, RabbitMQ, Jaeger, the OTEL collector, Prometheus, Grafana, and the core Go / web services using the sample environment contract.

After Postgres is healthy, apply migrations from the repository root:

```bash
export DATABASE_URL='postgres://healthai:healthai@localhost:5432/healthai?sslmode=disable'
make migrate-up
```

### Option B: GitHub Codespaces

Codespaces supports **Docker Compose** (lighter) and optional **kind + Tilt** (full local Kubernetes). The [`.devcontainer/`](../.devcontainer/) config installs Docker-in-Docker, Go, Node, kubectl, kind, Tilt, and `migrate`.

Prefer **8-core / 16GB+ RAM / 64GB storage** for Compose; for kind+Tilt prefer **8-core / 32GB+ storage**.

#### B1. Docker Compose (recommended)

1. On GitHub: **Code → Codespaces → Create codespace on …**.
2. Wait for the post-create step.
3. Start the stack:

```bash
./scripts/codespaces/prepare-env.sh
docker compose \
  -f deploy/docker/docker-compose.yml \
  -f deploy/docker/docker-compose.override.codespaces.yml \
  up --build
```

4. Apply migrations:

```bash
export DATABASE_URL='postgres://healthai:healthai@localhost:5432/healthai?sslmode=disable'
make migrate-up
```

5. In the **Ports** panel, set **visibility to Public** for `3000`, `8081`, and `8091` when opening `*.app.github.dev` URLs.
6. Open the forwarded web URL (`https://<codespace-name>-3000.app.github.dev`).

#### B2. kind + Tilt (Kubernetes in Docker)

Runs a local Kubernetes cluster inside the Codespace (kind), then starts the same Tilt stack used on a laptop.

```bash
./scripts/codespaces/kind-up.sh
# edit deploy/kubernetes/development/secrets.yaml — replace REPLACE_* values
./deploy/tilt/preflight.sh
tilt up
```

Open Tilt UI on forwarded port `10350`, and app ports as Tilt forwards them (`3000`, `8081`, …). Tear down with:

```bash
./scripts/codespaces/kind-down.sh
```

Do **not** run Compose and kind+Tilt at the same time (port and resource conflicts).

VS Code tasks under **Terminal → Run Task…**:

- `Codespaces: prepare env` / `compose up` / `migrate-up`
- `Codespaces: kind up` / `tilt up` / `kind down`

Voice/LiveKit and other provider features still need real API keys (see Troubleshooting).

### Option C: Tilt on local Kubernetes

Requires a local cluster (Docker Desktop Kubernetes, Minikube, or kind), plus `kubectl`, `tilt`, and `migrate`.

```bash
cp deploy/kubernetes/development/secrets.example.yaml deploy/kubernetes/development/secrets.yaml
# fill every placeholder, then:
./deploy/tilt/preflight.sh
tilt up
```

Tilt applies everything in `deploy/kubernetes/development/` (via Kustomize), including **Postgres, Redis, RabbitMQ, and Jaeger**, plus:

- **`db-migrate`** (label **database**) — runs `make migrate-up` when `DATABASE_URL` is set and `migrations/` changes. Requires the [`migrate`](https://github.com/golang-migrate/migrate) CLI on your laptop and a URL pointing at **`localhost:5432`** (Tilt port-forwards Postgres). Example:

```bash
export DATABASE_URL='postgres://healthai:YOUR_POSTGRES_PASSWORD@localhost:5432/healthai?sslmode=disable'
tilt up
```

If `DATABASE_URL` is empty, the **db-migrate** resource fails until you export it (same password as in `secrets.yaml`).

Application services Tilt builds and deploys:

- PostgreSQL
- Redis
- RabbitMQ
- Jaeger
- API gateway
- Auth service
- Patient service
- Notification service
- Health records service
- Analytics service
- Location service
- MCP server and voice-agent service
- Web and mobile frontends

**Not enabled in Tilt:** translation-model / translation-service (large LLM PVC; manifests exist but are omitted from `kustomization.yaml`). Legacy manifests (`rag-service`, `medical-records-service`) are also omitted.

## Common commands

```bash
make generate-proto
make migrate-up
make sqlc
kubectl get pods
```

## Access points

- Web app: `http://localhost:3000`
- API gateway (REST): `http://localhost:8081`
- Location service (patient WebSocket): `ws://localhost:8091` → path `/ws/location` (Tilt forwards host `8091` to container `8090`; voice-agent uses host `8090`)
- Jaeger UI: `http://localhost:16686`
- RabbitMQ management: `http://localhost:15672` (credentials from `rabbitmq-credentials` in `secrets.yaml`)
- Postgres: `localhost:5432` (via Tilt port-forward; see `postgres-secret` in `secrets.yaml`)
- Redis: `localhost:6379`
- Tilt UI: `http://localhost:10350`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3001`

The web app reads `NEXT_PUBLIC_API_URL` and `NEXT_PUBLIC_LOCATION_WS_URL` at **build** time. For Tilt, those are set in the `web` image build (`Tiltfile` + `deploy/docker/development/web.Dockerfile`). For local `npm run dev`, copy `web/.env.example` to `web/.env.local`.

**Location during voice calls (dev):** Keep the patient portal open with active `LOCATION_ACCESS`. On `start_location`, the browser should prompt for GPS; coordinates are sent over the WebSocket with `method: gps`. If GPS is denied or unavailable, the app calls `GET /v1/patient/location/approximate` on location-service for IP fallback (`method: ip-geolocation`) — that fallback does not work through Tilt port-forward (`127.0.0.1`), so use real GPS in local dev.

## Troubleshooting

### Secrets-related startup failures

- Re-check `deploy/kubernetes/development/secrets.yaml`
- Keep the same PostgreSQL password in both `postgres-secret` and `app-secrets.DATABASE_URL`

### Migration issues

- Ensure `migrate` is installed locally
- Run commands from the repository root

### Provider-dependent features

Some services require external credentials or separately managed infrastructure:

- LiveKit and SIP stack
- OpenAI
- Deepgram
- ElevenLabs
- Mailtrap
- VoIP.ms

Those integrations are part of the current implementation but are not yet fully abstracted behind provider interfaces.

### Secrets loading

Services can now use the shared `shared/secrets` package to prefer mounted secret files and fall back to environment variables. This keeps local Compose, local Kubernetes, and managed environments on the same lookup contract.

## Demo-friendly feature checks

After the stack is running, you can now exercise:

- patient login and the patient home page
- consent grant and revoke flows from the patient portal
- patient record question answering from the patient portal
- hospital summary generation plus the incident inbox
- nearest-hospital lookup through the location service

The roadmap-oriented reference docs for these additions are:

- `docs/product-purpose.md`
- `docs/react-native-mvp.md`
- `docs/phase3-rag-slice.md`
- `scripts/seed-data/README.md`
- `scripts/seed-fhir-data/README.md`
- `scripts/evaluation/README.md`

## FHIR sample data and external toolkits

### Seed synthetic FHIR into the dev database

1. Port-forward or expose `health-records-service` gRPC (`50054`).
2. Obtain the internal patient UUID you want to attach records to.
3. Run:

```bash
export INTERNAL_SERVICE_SECRET=your-dev-secret
go run ./scripts/seed-fhir-data \
  -patient-id "<uuid>" \
  -bundle examples/sample-fhir-data/demo-patient-bundle.json
```

The CLI uses the same `IngestFHIRBundle` path as hospital imports.

### Generate larger bundles with Synthea

[Synthea](https://github.com/synthetichealth/synthea) can export FHIR R4 bundles for synthetic patients. Export a single patient bundle, place it under `examples/sample-fhir-data/`, and re-run the seed command.

### Cross-check with HAPI FHIR (optional)

Run a [HAPI FHIR](https://hapifhir.io/) server locally to validate resources before ingestion:

```bash
docker run -p 8080:8080 hapiproject/hapi:latest
```

Upload a sample resource or bundle to HAPI, then ingest the same JSON into Zorba via `IngestFHIRBundle` or the seed CLI.

### Validate sample bundles in CI

```bash
go test ./services/health-records-service/internal/fhir/... -count=1
```

### Postman / HTTP samples

Import [`examples/sample-requests/zorba-health.postman_collection.json`](examples/sample-requests/zorba-health.postman_collection.json) for gateway-facing flows; use `grpcurl` for `HealthRecordService/IngestFHIRBundle` when testing ingestion directly.
