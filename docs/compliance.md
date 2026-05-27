# Compliance

Zorba Health now includes a dedicated Phase 2 compliance foundation centered on append-only auditing, explicit consent records, PHI-aware tool access, and emergency escalation tracking.

## Current baseline

- `services/audit-service` owns append-only audit writes and consent records in the `audit` schema
- `audit.audit_events` stores compliance events and `audit.consents` stores grant/revoke history
- the legacy `mcp_audit_log` table still exists as a compatibility path during migration
- `services/mcp-server` now emits richer audit events and checks consent before sensitive tool execution
- `services/voice-agent-service` contains the active safety/escalation logic for the current LiveKit-based agent runtime

## Audit versus analytics

- Audit is the compliance source of truth
- Analytics is for aggregate operational insight
- Analytics should avoid raw patient content, raw transcripts, and direct PHI exposure
- New compliance-sensitive features should write to audit first and only project analytics-safe aggregates later

## Audit event catalog

The canonical event catalog is implemented in `shared/audit/eventtypes.go` and mirrored in the database catalog table:

- `PATIENT_CREATED`
- `PATIENT_VERIFIED`
- `PATIENT_LOGIN`
- `PATIENT_LOGOUT`
- `HEALTH_RECORD_CREATED`
- `HEALTH_RECORD_VIEWED`
- `HEALTH_RECORD_SEARCHED`
- `HEALTH_RECORD_SUMMARIZED`
- `AI_TOOL_CALLED`
- `AI_RESPONSE_GENERATED`
- `LOCATION_REQUESTED`
- `EMERGENCY_ESCALATION_TRIGGERED`
- `NOTIFICATION_SENT`
- `CONSENT_GRANTED`
- `CONSENT_REVOKED`
- `TRANSLATION_REQUESTED`

## Consent model

Consent is now represented in `audit.consents` with grant/revoke history rather than a simple boolean flag.

Consent types currently modeled:

- `VOICE_ASSISTANT_USE`
- `HEALTH_RECORD_ACCESS`
- `LOCATION_ACCESS`
- `SMS_NOTIFICATION`
- `EMAIL_NOTIFICATION`
- `AI_SUMMARIZATION`
- `THIRD_PARTY_MODEL_PROCESSING`

Sensitive access paths should check consent before proceeding, especially:

- health-record retrieval
- AI summarization
- location access
- third-party model processing

## MCP compliance gateway

The MCP boundary now performs the core compliance checks before sensitive tool execution:

- auth verification
- actor/scope checks
- consent checks for PHI-sensitive tools
- audit start/completion logging with a shared correlation ID
- compatibility dual-write to the legacy `mcp_audit_log` path during migration

## Safety and escalation

The current voice runtime is `services/voice-agent-service`, so safety and escalation live there rather than in the removed Go worker service.

- obvious emergency phrases are treated as escalation triggers
- escalation attempts are recorded through the MCP `log_escalation` tool
- notification-service consumes escalation events and can fan out SMS notices
- emergency handling should avoid casual conversation once escalation is active

## Retention expectations

The architecture direction remains:

- temporary Redis-backed session state expires automatically
- location sessions expire after the related call or session
- transcripts should default to disabled unless intentionally enabled
- audit data should follow explicit retention policy rather than ad hoc deletion

## Contributor expectations

- document any feature that introduces PHI handling
- note whether new features belong to analytics, audit, or both
- prefer appending new audit events over mutating historical compliance records
- avoid adding features that blur compliance and product analytics without documenting the rationale
