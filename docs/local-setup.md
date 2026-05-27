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

```bash
tilt up
```

Tilt currently builds and applies:

- PostgreSQL
- Redis
- RabbitMQ
- API gateway
- Auth service
- Patient service
- Notification service
- Health records service
- Analytics service
- Location service
- Translation model and translation service
- Agent worker service
- Web frontend

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
- Tilt UI: `http://localhost:10350`

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
