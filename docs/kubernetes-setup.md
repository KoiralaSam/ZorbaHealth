# Kubernetes Setup

Zorba Health currently ships Kubernetes manifests under `deploy/kubernetes/`.

## Current manifest layout

- Development manifests: `deploy/kubernetes/development`
- Production-oriented manifests: `deploy/kubernetes/production`

## Local Kubernetes workflow

1. Enable a local Kubernetes cluster.
2. Copy `deploy/kubernetes/development/secrets.example.yaml` to `deploy/kubernetes/development/secrets.yaml`.
3. Fill real values for database, JWT, email, SMS, and AI provider secrets.
4. Run `tilt up`.

Tilt applies development manifests for:

- ConfigMap
- Secrets
- PostgreSQL
- Redis
- RabbitMQ
- application services

## Namespaces and ingress

The current development manifests are optimized for local use and do not yet provide a production-grade namespace, ingress, or HPA strategy. Those are planned follow-up tasks in later phases.

## Resource management

Current manifests are sufficient for local development, but they should later be expanded with:

- explicit resource requests and limits
- Horizontal Pod Autoscaler policies
- ingress resources
- stronger secret-management integration
- service-to-service TLS and identity controls

## Production note

The `deploy/kubernetes/production` directory should be treated as a starting point, not a finished platform package. Contributors preparing production deployments should also review:

- `docs/deployment.md`
- `docs/security.md`

## Future Helm support

The planned open-source target layout includes `deploy/helm`, but Helm charts are not yet implemented in the current repository.
