# Security

This document records the current security posture and the planned hardening direction for Zorba Health.

## Current baseline

- JWT secrets are configured through environment or Kubernetes secrets
- patient and auth services already use JWT-related secrets
- local development uses Kubernetes secrets from `deploy/kubernetes/development/secrets.example.yaml`
- several services communicate internally over gRPC
- external providers are accessed with API keys

## Session and refresh tokens (implemented)

- Login creates an `auths` session row (family root) plus a rotating opaque refresh token (SHA-256 hash stored in `refresh_tokens`).
- Access JWTs are short-lived (default 15 minutes, `ACCESS_TOKEN_TTL`).
- Refresh rotation uses automatic reuse detection: presenting a previously used refresh token revokes the entire family and returns `REFRESH_TOKEN_REUSE`.
- Web clients store refresh tokens in HttpOnly cookies on the API origin (`credentials: "include"`); mobile stores refresh tokens in SecureStore and sends `X-Zorba-Client: mobile` for JSON refresh tokens on login.

## Current limitations

- Redis-backed `jti` denylist for access tokens is not wired yet (family revocation covers refresh theft)
- no mTLS between services yet
- service identity is still shared-secret based by default in development

## Canonical actor and permission model

The repository uses these actor classes for authorization decisions:

| Actor | Token actor type | Primary identifiers | Allowed baseline actions |
| --- | --- | --- | --- |
| Patient | `patient` | `patientID`, `sessionID` | access own portal profile, consent records, call summaries, audit trail, and consent-gated record Q&A |
| Provider staff | `staff` | `staffID`, `hospitalID`, `role` | access hospital incident queue, consented patient summaries, and consented patient audit views for their hospital |
| Admin | `admin` | `adminID` | operate administrative surfaces that are explicitly marked admin-only |
| System service | internal secret or future service identity | service name | call internal gRPC APIs required for workflow execution |
| Auditor | planned role | auditor ID, organization scope | read compliance evidence without mutating patient care data |

Authorization defaults:

- fail closed when actor type or required identifier is missing
- require patient tokens for patient portal routes
- require staff tokens with `hospitalID` for hospital routes
- require explicit consent for PHI-sensitive patient record and provider summary workflows
- audit denied and successful access paths where PHI or consent state is involved
- keep system-service privileges separate from patient and staff bearer tokens

## Token, CORS, and revocation policy

Current access tokens are JWTs verified through shared helpers. The target design is:

- short-lived access tokens
- rotating refresh tokens stored server-side
- Redis-backed revocation keyed by token ID or session ID
- logout revokes the active refresh session and blocks reuse
- internal service calls use service identity rather than long-lived user tokens when the action is not on behalf of a user

API gateway CORS is configured with `API_GATEWAY_ALLOWED_ORIGINS`, a comma-separated list of exact trusted origins. Credentialed healthcare routes must not use wildcard origins. Browser preflight requests from untrusted origins are rejected.

## Bridged-call interpretation controls

- Patient and staff translation controls are mediated by API Gateway and patient-service rather than being written directly from clients to Redis.
- Session-scoped interpretation preferences are stored in Redis with automatic expiration and should never be treated as a source of truth for historical compliance review.
- Translation-service uses `TRANSLATION_PROVIDER=amazon_translate` plus AWS credentials or IRSA at runtime; do not embed AWS keys in source, mobile bundles, or web build arguments.
- Bridged-call control APIs must authorize both actor identity and hospital scope before returning session state or mutating translation preferences.

## Security principles for contributors

- never commit secrets
- never log full patient names, phone numbers, emails, addresses, raw transcripts, or complete medical records
- prefer request IDs, correlation IDs, service names, statuses, and hashed identifiers in logs
- keep external provider credentials isolated in environment variables or secret stores
- LiveKit credentials are reachable from **patient-service only**; use `LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET`, and `LIVEKIT_PUBLIC_WS_URL` from the runtime secret store. The voice agent and Zorba MCP must not expose meeting creation tools, and patient-facing channels receive join links only after staff approval.

## PHI-safe logging standard

The shared helper in `shared/logging/safe.go` is the default path for hot logging paths that might otherwise expose PHI.

Allowed log fields:

- request IDs
- correlation IDs
- service names
- event types
- status / duration
- hashed identifiers such as hashed phone numbers or emails

Voice OTP channels (DTMF, inbound SMS, spoken):

- Log session IDs, verification correlation IDs, channel enums, and hashed phone identifiers only.
- Never log OTP values, inbound SMS bodies, or per-digit DTMF keys in traces or application logs.
- MCP and patient-service reject verification when JWT `callerPhone` does not match the requested phone.

Disallowed log content:

- raw phone numbers
- raw email addresses
- raw SMS bodies or transcripts
- raw translated segments
- full patient names
- addresses
- full medical notes or record blobs

When in doubt, hash identifiers and log only operational metadata.

## Secrets handling

Current local development path:

- `deploy/kubernetes/development/secrets.example.yaml`
- `deploy/kubernetes/development/secrets.yaml` for local-only secret material

Shared loader package:

- `shared/secrets` now provides a common `SecretStore` abstraction.
- Supported local backends today: environment variables and mounted secret files.
- Docker and Kubernetes secret mounts can both use the file-backed store.
- Cloud backends are stubbed behind the same interface: Vault, AWS Secrets Manager, Google Secret Manager, and Azure Key Vault.

Recommended lookup order for services:

1. mounted secret file (`/run/secrets` or Kubernetes projected volume)
2. environment variable override for local development
3. cloud secret store only in managed environments

Field-level encryption planning:

- `shared/secrets.KMSProvider` is the contract for future managed encryption keys
- minimum PHI columns to consider for envelope encryption: patient email, patient phone number, and any future persistent raw transcript storage
- hashes remain preferable for lookup-only use cases such as phone normalization and audit-safe identifiers

Planned future support:

- Docker secrets
- Vault
- AWS Secrets Manager
- Google Secret Manager
- Azure Key Vault

## Internal gRPC security modes

Sensitive internal gRPC surfaces include:

- `audit-service`
- `health-records-service`

The shared helper package is `shared/grpc/auth`.

Development mode:

- transport may remain insecure inside the local cluster
- every request must carry `x-internal-token`
- services may also send `x-internal-service` for identity attribution

Production target mode:

- mutual TLS for all sensitive service-to-service calls
- internal token or signed internal JWT layered on top of mTLS for authorization
- allow-lists on sensitive servers for expected callers

To generate local test certificates, use:

```bash
./scripts/security/generate-dev-mtls-certs.sh
```

This script is for local experimentation only; production certificates should come from cluster PKI such as cert-manager or SPIFFE/SPIRE.

## Threat model notes

High-priority threats:

- stolen refresh tokens or long-lived browser sessions
- PHI leakage through logs, dashboards, traces, and message retries
- over-broad internal service privileges
- persistent storage of precise patient location beyond call scope
- provider credential sprawl across services

Current mitigations:

- rotating refresh sessions
- PHI-safe logging helpers
- consent-aware MCP gateway and audit trails
- ephemeral location/session patterns
- provider configuration isolated to env/secrets rather than source

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
