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
- API gateway: `http://localhost:8081`
- Tilt UI: `http://localhost:10350`

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
- SendGrid
- VoIP.ms

Those integrations are part of the current implementation but are not yet fully abstracted behind provider interfaces.
