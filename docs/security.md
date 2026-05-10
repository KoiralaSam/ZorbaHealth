# Security

This document records the current security posture and the planned hardening direction for Zorba Health.

## Current baseline

- JWT secrets are configured through environment or Kubernetes secrets
- patient and auth services already use JWT-related secrets
- local development uses Kubernetes secrets from `deploy/kubernetes/development/secrets.example.yaml`
- several services communicate internally over gRPC
- external providers are accessed with API keys

## Current limitations

- no centralized refresh-token and revocation design is documented yet
- no uniform role and policy model is fully documented at the repository level
- no repo-wide PHI-safe logging standard is codified yet
- no mTLS between services yet
- no formal audit/compliance service yet
- no standardized provider abstraction layer yet

## Security principles for contributors

- never commit secrets
- never log full patient names, phone numbers, emails, addresses, raw transcripts, or complete medical records
- prefer request IDs, correlation IDs, service names, statuses, and hashed identifiers in logs
- keep external provider credentials isolated in environment variables or secret stores

## Secrets handling

Current local development path:

- `deploy/kubernetes/development/secrets.example.yaml`
- `deploy/kubernetes/development/secrets.yaml` for local-only secret material

Planned future support:

- Docker secrets
- Vault
- AWS Secrets Manager
- Google Secret Manager
- Azure Key Vault

## Planned hardening roadmap

- short-lived JWT access tokens
- refresh token strategy
- Redis-backed token revocation
- role model covering `PATIENT`, `PROVIDER`, `ADMIN`, `SYSTEM_SERVICE`, and `AUDITOR`
- future ABAC expansion
- mTLS and service identity for internal gRPC calls
- stronger webhook signature verification
- field-level encryption for especially sensitive PHI

## Related documents

- `docs/compliance.md`
- `docs/kubernetes-setup.md`
- `docs/local-setup.md`
