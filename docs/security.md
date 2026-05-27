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
- no mTLS between services yet
- no standardized provider abstraction layer yet

## Security principles for contributors

- never commit secrets
- never log full patient names, phone numbers, emails, addresses, raw transcripts, or complete medical records
- prefer request IDs, correlation IDs, service names, statuses, and hashed identifiers in logs
- keep external provider credentials isolated in environment variables or secret stores

## PHI-safe logging standard

The shared helper in `shared/logging/safe.go` is the default path for hot logging paths that might otherwise expose PHI.

Allowed log fields:

- request IDs
- correlation IDs
- service names
- event types
- status / duration
- hashed identifiers such as hashed phone numbers or emails

Disallowed log content:

- raw phone numbers
- raw email addresses
- raw SMS bodies or transcripts
- full patient names
- addresses
- full medical notes or record blobs

When in doubt, hash identifiers and log only operational metadata.

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
- broader automated checks to catch unsafe log patterns in CI

## Related documents

- `docs/compliance.md`
- `docs/kubernetes-setup.md`
- `docs/local-setup.md`
