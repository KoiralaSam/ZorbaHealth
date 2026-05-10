# Deployment

This document describes the current deployment posture and the expected future open-source packaging direction.

## Current state

The repository currently supports:

- local development with Tilt and Kubernetes
- development Dockerfiles under `deploy/docker/development`
- production-oriented Dockerfiles under `deploy/docker/production`
- production-oriented Kubernetes manifests under `deploy/kubernetes/production`

The repository does not yet provide:

- Docker Compose for local full-stack startup
- Helm charts
- a fully standardized production deployment story

## Local deployment path

Use:

```bash
tilt up
```

This is the primary supported developer workflow today.

## Production-oriented path

The current production assets are found under:

- `deploy/docker/production`
- `deploy/kubernetes/production`

These should be treated as a base for environment-specific adaptation.

## External infrastructure

The following components are expected to run outside the repository-managed local Kubernetes stack:

- FreePBX
- LiveKit
- LiveKit SIP
- cloud or self-hosted AI providers

## Planned deployment maturity improvements

- Docker Compose for open-source onboarding
- Helm chart packaging
- resource requests and limits
- autoscaling policies
- ingress management
- stronger secrets integration
- CI-driven image builds and validation
