# Deployment

This document describes the current deployment posture.

## Current state

The repository supports:

- **Docker Compose** for local and Codespaces full-stack startup (`deploy/docker/`)
- **Tilt + local Kubernetes** for cluster-shaped development (`Tiltfile`, `deploy/kubernetes/development/`)
- Optional **Helm** packaging under `deploy/helm/` with `deploy/helm/values/dev.yaml`
- Development and production-oriented Dockerfiles under `deploy/docker/`
- Production-oriented Kubernetes manifests under `deploy/kubernetes/production`

AWS EKS / ECR deploy scripts and the former EKS GitHub Actions workflow have been removed.

## Local deployment path (recommended)

```bash
docker compose \
  -f deploy/docker/docker-compose.yml \
  -f deploy/docker/docker-compose.override.local.yml \
  up --build
```

See [`docs/local-setup.md`](local-setup.md) for Codespaces and optional Tilt.

## Optional Helm

```bash
helm dependency build deploy/helm/
helm upgrade --install zorbahealth deploy/helm/ \
  --namespace dev \
  -f deploy/helm/values/dev.yaml \
  --create-namespace
```

Do not run Helm and Tilt against the same `dev` namespace at once.

## Production-oriented path

The current production assets are found under:

- `deploy/docker/production`
- `deploy/kubernetes/production`

These should be treated as a base for environment-specific adaptation.

## External infrastructure

The following components are expected to run outside the repository-managed local stack:

- FreePBX
- LiveKit
- LiveKit SIP
- cloud or self-hosted AI providers
