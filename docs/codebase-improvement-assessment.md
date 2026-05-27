# Codebase Improvement Assessment

Date: 2026-05-27

This document summarizes what appears to be working, what is still thin or not working, and how to improve it based on the current plans, docs, and code structure. It intentionally does not make code changes.

## Evidence Reviewed

- `README.md`
- `docs/architecture.md`
- `docs/service-map.md`
- `docs/product-purpose.md`
- `docs/phase3-rag-slice.md`
- `docs/react-native-mvp.md`
- `docs/security.md`
- `docs/compliance.md`
- `docs/evaluation.md`
- `docs/q1-journal-evaluation-plan.md`
- `docs/local-setup.md`
- `docs/deployment.md`
- `docs/pgvector-indexing.md`
- `docs/open-source-extension-guide.md`
- `docs/architecture/voice-call-flow-v1.md`
- `docs/architecture/rabbitmq-flow-v1.md`
- `services/health-records-service/README.md`
- `services/voice-agent-service/README.md`
- selected code in `services/api-gateway`, `services/health-records-service`, `services/mcp-server`, `services/location-service`, `services/health-provider-service`, `web`, and `mobile`
- Graphify query over the existing code knowledge graph

## Executive Summary

The repo has moved beyond a concept prototype. Several core backend foundations are present: Go microservices, gRPC contracts, RabbitMQ events, PostgreSQL migrations, audit and consent service boundaries, MCP tool gating, LiveKit voice-agent wiring, patient and hospital web surfaces, health-record RAG code, FHIR ingestion modules, and an Expo mobile app shell.

The biggest improvement opportunity is not adding more isolated features. The next step should be making the implemented paths reproducible, testable, and trustworthy end to end. The docs describe a strong product and research direction, but multiple areas still need integration hardening: provider abstraction, evaluation scripts, deployment packaging, role/security policy, service identity, health-provider APIs, production-ready location lookup, and full smoke coverage across patient, voice, hospital, records, consent, audit, and notification flows.

## What Is Working

### 1. Repository and Service Structure

The top-level layout is now coherent for an open-source microservice project:

- `services/` contains service implementations.
- `proto/` contains source protobuf contracts.
- `shared/proto/` contains generated Go bindings.
- `migrations/` contains database migrations.
- `deploy/` contains Docker and Kubernetes assets.
- `docs/`, `examples/`, and `scripts/` provide contributor-facing support.

This is a good base for contributors because service ownership and deployment assets are no longer hidden inside ad hoc folders.

What to preserve:

- Keep source `.proto` files under `proto`.
- Keep generated protobuf output under `shared/proto`.
- Keep service-specific logic in each service's `internal/` boundary.
- Keep deployment manifests under `deploy/`.

### 2. Core Backend Service Boundaries

The current service map shows a meaningful microservice split:

- API gateway for HTTP-facing web and mobile APIs.
- Auth service for account and token behavior.
- Patient service for registration, verification, and patient identity flows.
- Health records service for FHIR storage, vector chunks, retrieval, and summarization.
- MCP server as the controlled tool gateway for voice and agent workflows.
- Audit service for append-only audit and consent.
- Voice agent service for LiveKit, STT, LLM, TTS, VAD, and turn handling.
- Notification service for patient and emergency messaging.
- Location service for live location and nearest-hospital lookup.
- Translation and analytics services for supporting capabilities.

The code also reflects this split. For example, the API gateway registers patient, hospital, consent, records-answer, call-summary, audit, and incident endpoints in `services/api-gateway/cmd/api-gateway/main.go`, and the MCP gateway enforces auth, scopes, consent, and audit in `services/mcp-server/internal/gateway/gateway.go`.

What to preserve:

- Keep MCP as the boundary between voice-agent tool calls and backend services.
- Keep audit/consent out of random feature handlers and centralized in audit-service plus MCP/records enforcement.
- Keep the API gateway as the stable HTTP surface for web and mobile.

### 3. Compliance Foundation

The compliance docs and code describe a solid Phase 2 baseline:

- `services/audit-service` owns append-only audit events and consent records.
- Consent types are modeled explicitly.
- MCP tools perform auth, scope, consent, and audit checks.
- Health-record RAG checks `HEALTH_RECORD_ACCESS` before retrieval.
- Emergency escalation events are modeled and routed toward notifications.

This is one of the more important architectural strengths because healthcare AI features become risky quickly if audit and consent are bolted on later.

What to preserve:

- Append audit events rather than mutating historical compliance records.
- Treat audit as the source of truth, separate from analytics.
- Keep PHI-sensitive tools consent-gated.
- Keep raw transcripts and raw records out of analytics by default.

### 4. Health Records, FHIR, and RAG Direction

The Phase 3 RAG docs now map to actual implementation areas:

- `services/health-records-service/internal/fhir/` for validation, normalization, and bundle ingestion.
- `services/health-records-service/internal/rag/` for consent-aware retrieval, reranking, and summarization.
- `records.fhir_resources`, `records.fhir_patient_map`, and `records.record_chunks` are described in migrations and docs.
- pgvector is used with OpenAI `text-embedding-3-small` dimensions.
- RAG results are expected to include citations.

The current `internal/rag/pipeline.go` already performs patient ID validation, consent checks, embedding, vector candidate search, metadata filtering, lightweight reranking, summarization over retrieved chunks only, and audit logging.

What to preserve:

- Keep summarization grounded in retrieved chunks only.
- Return source metadata such as `chunk_id`, `record_id`, and `fhir_resource_type`.
- Keep FHIR ingestion inside health-records-service.
- Keep patient filters on vector searches for privacy and query planning.

### 5. Voice Agent Direction

The voice-agent service has a clear role:

- LiveKit Agents worker.
- STT through Deepgram.
- LLM through OpenAI.
- TTS through Deepgram or ElevenLabs.
- MCP-backed tools for backend calls.
- LiveKit webhook endpoint and health endpoint.

The current voice flow doc also makes the right distinction between patient voice sessions and staff summary workflows. Voice record access requires verified patient identity and scoped patient tokens, while staff summaries go through hospital-facing API paths.

What to preserve:

- Keep direct backend access out of the voice agent.
- Continue to use MCP tools for patient lookup, verification, records answers, escalation, translation, and location.
- Keep safety and escalation logic in the current active runtime until it is formally extracted.

### 6. Web and Mobile Product Surfaces

The web app has routes for patient login, registration, patient portal, hospital login, and hospital records summary. The API contracts include patient profile, consent, records answer, calls, audit, hospital incidents, and hospital patient audit paths.

The mobile app exists as an Expo React Native app and its README claims support for:

- patient login and registration
- OTP and email verification
- profile
- consent center
- health-record Q&A
- call summaries
- audit trail
- emergency location sharing
- hospital login and hospital workflows

This is a strong product direction because the mobile app reuses the same gateway endpoints instead of introducing a separate backend.

What to preserve:

- Keep web and mobile on the same API gateway contract.
- Keep in-app voice deferred until the call state machine and agent lifecycle are stronger.
- Keep location WebSocket access scoped and consent-gated.

## What Is Not Working or Still Thin

### 1. Health Provider Service Is Still a Stub

Docs explicitly describe health-provider functionality as future work, and `services/health-provider-service/cmd/health-provider-service/main.go` currently only creates an empty HTTP mux with a TODO for provider registration and organization profile handlers.

Why this matters:

- Hospital workflows need real provider, organization, facility, and staff models.
- Staff auth and patient access policies eventually need organization context.
- Audit queries need a reliable provider/staff identity model.

Recommended implementation path:

1. Define provider-domain models:
   - `Organization`
   - `Facility`
   - `Provider`
   - `StaffMembership`
   - `ProviderPatientRelationship`
2. Add protobuf contracts under `proto/health_provider.proto` or `proto/provider/*.proto` if the service should be gRPC.
3. Add migrations for provider schema:
   - `provider.organizations`
   - `provider.facilities`
   - `provider.staff_memberships`
   - `provider.patient_relationships`
4. Implement service structure matching other Go services:
   - `cmd/health-provider-service`
   - `internal/core/domain/models`
   - `internal/core/ports/inbound`
   - `internal/core/ports/outbound`
   - `internal/core/services`
   - `internal/adapters/primary/http` or `grpc`
   - `internal/adapters/secondary/repositories/postgres`
5. Wire hospital-facing gateway routes to this service instead of encoding provider assumptions directly in API gateway handlers.

Suggested stack:

- Go
- PostgreSQL
- sqlc for typed queries
- gRPC for service-to-service calls
- HTTP only at API gateway unless there is a strong reason to expose provider-service directly

### 2. Provider Abstraction Is Incomplete

Docs call out missing provider-agnostic abstractions for LLM, embeddings, STT, TTS, email, SMS, telephony, translation, vector search, and FHIR. Current code already has some local interfaces, such as health-records outbound ports and translation service client boundaries, but the overall system still couples directly to concrete providers:

- OpenAI for embeddings/summarization.
- Deepgram for STT.
- ElevenLabs or Deepgram for TTS.
- LiveKit for realtime voice.
- Mailtrap and VoIP.ms for notifications.
- Nominatim/OpenStreetMap for hospital lookup.
- translation-model HTTP backend for translation.

Why this matters:

- Open-source users need to swap providers.
- Research evaluation should distinguish architecture from a single vendor stack.
- Healthcare deployments often have provider and data-processing restrictions.

Recommended implementation path:

1. Standardize outbound provider ports per service, not globally at first:
   - `Embedder`
   - `Summarizer`
   - `SpeechToText`
   - `TextToSpeech`
   - `NotificationSender`
   - `SMSProvider`
   - `EmailProvider`
   - `TelephonyProvider`
   - `TranslationProvider`
   - `HospitalFinder`
2. Move concrete SDK/API clients into `internal/adapters/secondary/<provider>`.
3. Use config-driven provider selection:
   - `EMBEDDING_PROVIDER=openai`
   - `SUMMARIZER_PROVIDER=openai`
   - `STT_PROVIDER=deepgram`
   - `TTS_PROVIDER=elevenlabs`
   - `SMS_PROVIDER=voipms`
   - `EMAIL_PROVIDER=mailtrap`
4. Keep provider-specific environment variables documented in `examples/sample-env/.env.example` and `docs/local-setup.md`.
5. Add fake providers for local smoke tests:
   - deterministic embedder
   - deterministic summarizer
   - no-send email/SMS provider
   - local translation echo provider

Suggested stack:

- Go interfaces for backend services.
- Python protocols or small adapter classes for `voice-agent-service`.
- Dependency injection at service startup.
- Contract tests for each provider interface.

### 3. Evaluation Harness Is Mostly Planned, Not Executable

`docs/evaluation.md`, `docs/q1-journal-evaluation-plan.md`, and `scripts/evaluation/README.md` define the right scenarios and metrics, but the repo still needs executable scripts for:

- patient portal smoke
- consent gating
- patient record Q&A
- hospital escalation smoke
- nearest-hospital smoke
- RAG groundedness checks
- latency and reproducibility measurements

Why this matters:

- The project aims at a journal-quality evaluation narrative.
- Claims around safety, grounding, latency, and reproducibility need executable evidence.
- Contributors need a quick way to verify that changes did not break the product path.

Recommended implementation path:

1. Add a small evaluation CLI under `scripts/evaluation`.
2. Use scenarios from `scripts/evaluation/README.md` as executable commands:
   - `patient-portal-smoke`
   - `consent-gating-check`
   - `hospital-escalation-smoke`
   - `nearest-hospital-smoke`
   - `rag-groundedness-check`
3. Store results in machine-readable JSON:
   - status
   - latency
   - request ID
   - endpoint or gRPC method
   - assertion failures
4. Add seed prerequisites:
   - demo patient
   - consent grants
   - FHIR bundle
   - emergency escalation event
5. Keep early scripts HTTP/gRPC based before adding browser automation.

Suggested stack:

- Go for gRPC-heavy checks, because the repo already uses Go and generated protobufs.
- TypeScript/Node or Playwright for browser smoke once the HTTP checks are stable.
- JSON output for CI and later journal tables.

### 4. Deployment Packaging Is Developer-Only

The supported local path is currently Tilt plus Kubernetes. Production-oriented Dockerfiles and Kubernetes manifests exist, but the docs explicitly say there is no Docker Compose, Helm chart, standardized production deployment story, full resource policy, autoscaling, ingress, or secrets integration.

Why this matters:

- Open-source onboarding is harder if users need local Kubernetes immediately.
- Healthcare demos often need repeatable environment setup.
- Production claims are weak without resource limits, ingress, TLS, secret integration, and health checks.

Recommended implementation path:

1. Add Docker Compose for the minimum local demo:
   - PostgreSQL with pgvector
   - Redis
   - RabbitMQ
   - API gateway
   - auth-service
   - patient-service
   - audit-service
   - health-records-service
   - mcp-server
   - notification-service in no-send mode
   - location-service
   - web
2. Keep LiveKit, SIP, and real AI providers optional in Compose.
3. Add Helm charts after Compose stabilizes:
   - one chart per service or a parent chart with service subcharts
   - configurable image repositories
   - resource requests and limits
   - probes
   - ingress
   - secrets references
4. Add production deployment docs with environment-specific overlays.

Suggested stack:

- Docker Compose for contributor demo mode.
- Helm for Kubernetes packaging.
- Kustomize or Helm values for development versus production.
- GitHub Actions for image builds and manifest validation.

### 5. Security Model Needs Centralized Token, Role, and Revocation Design

`docs/security.md` identifies missing pieces:

- centralized refresh-token and revocation design
- uniform role and policy model
- mTLS or service identity
- stronger webhook verification
- field-level encryption for sensitive PHI
- automated unsafe-log checks

Code has JWT handling and claim verification in multiple places, including shared auth helpers and service-specific interceptors, but the policy model is not yet documented as one canonical source.

Why this matters:

- Patient, provider, admin, auditor, and system-service actors need consistent authorization.
- Long-lived access tokens without revocation create risk.
- Internal gRPC calls rely heavily on shared secrets and forwarded tokens.

Recommended implementation path:

1. Define a canonical actor and permission matrix:
   - `PATIENT`
   - `PROVIDER`
   - `ADMIN`
   - `SYSTEM_SERVICE`
   - `AUDITOR`
2. Add a security design doc for:
   - access-token TTL
   - refresh-token rotation
   - Redis-backed revocation
   - logout semantics
   - service-to-service auth
   - audit requirements for denied access
3. Centralize claim validation in `shared/auth` and reduce duplicate parsing.
4. Add middleware/interceptor tests for each role and denied path.
5. Add CI checks for unsafe logging patterns.
6. Move toward mTLS or SPIFFE/SPIRE-style service identity when deployment packaging is mature.

Suggested stack:

- JWT access tokens with short TTL.
- Rotating refresh tokens stored server-side.
- Redis revocation set keyed by token ID or session ID.
- Go gRPC interceptors for service authorization.
- HTTP middleware in API gateway.
- Static checks with `rg` patterns first, then custom linting if needed.

### 6. CORS and Environment Configuration Are Too Localhost-Specific

The API gateway CORS middleware currently hardcodes `http://localhost:3000`. This works for local web development but will fail for mobile, alternate local ports, preview URLs, and production domains unless proxied or changed.

Why this matters:

- The mobile app uses API gateway endpoints too.
- Deployed web environments need configurable origins.
- Hardcoded origins are a common source of confusing integration failures.

Recommended implementation path:

1. Introduce `API_GATEWAY_ALLOWED_ORIGINS`.
2. Parse a comma-separated origin list at startup.
3. Reflect only exact allowed origins.
4. Keep credentials enabled only for trusted origins.
5. Add tests for:
   - allowed origin
   - denied origin
   - preflight
   - missing origin

Suggested stack:

- Go HTTP middleware.
- Environment-driven config through existing `shared/env`.
- Table-driven Go tests.

### 7. Location and Emergency Flow Need Production Hardening

The docs previously called nearest-hospital lookup a noop/stub, while current code includes a `NominatimHospitalFinder` with seeded fallback hospitals. This is progress, but the flow is still not production-ready:

- Nominatim is not a healthcare-grade provider contract.
- Fallback hospitals are demo data.
- WebSocket location is separate from API gateway, which is fine, but needs clear auth, lifecycle, and observability.
- IP geolocation fallback does not work well through local loopback/Tilt.

Why this matters:

- Emergency escalation is safety-sensitive.
- Location sharing must be explicit, consented, and short-lived.
- Provider lookup needs reliability, rate limiting, and a fallback strategy.

Recommended implementation path:

1. Define `HospitalFinder` as a stable outbound interface.
2. Keep Nominatim as a demo adapter.
3. Add a production provider adapter later:
   - Google Places
   - Mapbox
   - healthcare facility registry
   - self-hosted facility database
4. Store seeded demo facilities in a database table instead of hardcoded fallback slices.
5. Add event tracing:
   - `call.started`
   - `start_location`
   - `location_update`
   - `nearest_hospital_requested`
   - `emergency_escalation_triggered`
6. Add expiration and cleanup tests for Redis location sessions.

Suggested stack:

- Go interface in location-service outbound ports.
- Redis TTL for live location state.
- PostgreSQL table for demo/provider facility data.
- RabbitMQ events for call lifecycle.
- OpenTelemetry spans around lookup and WebSocket lifecycle.

### 8. RAG Is Implemented but Needs Evaluation and Guardrails

The RAG pipeline is present, but docs still identify follow-on work:

- FHIR ingestion and normalization improvements
- pgvector-backed chunk search tuning
- stronger source-grounding evaluation
- richer patient-facing explanations of retrieved evidence

The current reranker is intentionally lightweight. It boosts exact query text and selected resource types, which is acceptable for an early deterministic slice but not enough for robust clinical-style retrieval.

Why this matters:

- Health-answering quality depends on retrieval quality.
- Patient answers need citations and should avoid overconfident conclusions.
- Journal claims need groundedness and hallucination measurements.

Recommended implementation path:

1. Add a small synthetic RAG QA set under `examples/` or `scripts/evaluation/fixtures`.
2. Add evaluation commands:
   - answer contains citation
   - answer refuses when no relevant records exist
   - answer respects consent denial
   - answer cites the expected resource type
3. Improve chunk metadata:
   - resource type
   - code text
   - effective date
   - source bundle/file
   - normalized patient linkage
4. Add reranking abstraction:
   - keep keyword/resource boost as default
   - allow later cross-encoder or LLM rerank adapters
5. Add answer policy:
   - cite sources
   - state uncertainty
   - avoid diagnosis
   - recommend contacting a clinician for urgent or unclear cases

Suggested stack:

- pgvector for ANN search.
- PostgreSQL btree filters on patient/resource metadata.
- Go table tests for deterministic retrieval.
- Optional LLM-judge only for offline evaluation, not runtime truth.

### 9. Mobile App Exists, but It Needs Verification Against the API Gateway Contract

The mobile README says the Expo app supports patient and hospital workflows, and `mobile/App.tsx` references the current gateway routes. The docs originally framed mobile as an MVP plan, while the code now appears to have a larger single-file implementation.

Why this matters:

- A single large `App.tsx` can become hard to test and maintain.
- Mobile security requirements are stricter around token storage and PHI caching.
- The location WebSocket path must be verified on real devices and simulators.

Recommended implementation path:

1. Split `mobile/App.tsx` into:
   - `src/api/client.ts`
   - `src/auth/tokenStore.ts`
   - `src/screens/PatientHome.tsx`
   - `src/screens/ConsentCenter.tsx`
   - `src/screens/HealthQuestion.tsx`
   - `src/screens/HospitalDashboard.tsx`
   - `src/location/useEmergencyLocationSession.ts`
2. Add typed API response models shared with or generated from backend contracts where practical.
3. Add Expo SecureStore tests/mocks.
4. Verify:
   - login
   - token persistence
   - consent list/mutation
   - records answer
   - call summaries
   - location WebSocket
5. Keep raw health-record data out of persistent storage.

Suggested stack:

- Expo
- React Native
- TypeScript
- `expo-secure-store`
- a small typed fetch wrapper
- Jest plus React Native Testing Library for component logic

### 10. Web App README and Frontend Verification Are Behind the Product

`web/README.md` is still mostly the default Next.js README. The web app itself has product routes, but the documentation does not explain the actual patient and hospital workflows.

Why this matters:

- Contributors will not know which flows are supported.
- CI and manual QA need route-level expectations.
- The frontend is now more than a default create-next-app shell.

Recommended implementation path:

1. Update `web/README.md` to document:
   - required env vars
   - patient routes
   - hospital routes
   - local gateway URL
   - location WebSocket URL
   - common dev commands
2. Add frontend smoke tests:
   - login page renders
   - patient page handles empty state
   - hospital summary form validates input
   - consent controls render with fixture data
3. Add API contract mocks for local component tests.

Suggested stack:

- Next.js App Router
- TypeScript
- Playwright for end-to-end smoke
- React Testing Library for local components
- MSW for HTTP mocks if component tests need API responses

### 11. RabbitMQ Docs Are Partly Outdated

`docs/architecture/rabbitmq-flow-v1.md` still references `Agent Worker Service`, `RAG Service`, and `Medical Records Service` naming that does not fully match the current service map, where the active runtime is `voice-agent-service`, RAG lives inside health-records-service, and health-records-service is the current records boundary.

Why this matters:

- Event architecture docs need to match actual service names and event producers.
- Incorrect diagrams cause integration mistakes.
- RabbitMQ events are central to patient lifecycle, notification, location, and escalation flows.

Recommended implementation path:

1. Update event flow docs to current names:
   - `voice-agent-service`
   - `health-records-service`
   - `mcp-server`
   - `notification-service`
   - `location-service`
   - `audit-service`
2. Add an event catalog in code and docs:
   - exchange
   - routing key
   - producer
   - consumers
   - payload schema
   - retry/dead-letter behavior
3. Add contract tests for event payload structs in `shared/events`.

Suggested stack:

- Go structs in `shared/events`.
- RabbitMQ exchange/queue declarations in `shared/messaging`.
- Mermaid docs generated or checked against event constants where possible.

### 12. Observability Is Started but Not Complete

The API gateway initializes OpenTelemetry tracing, and docs call out future OpenTelemetry coverage. The rest of the platform needs consistent tracing, logging, metrics, and correlation IDs.

Why this matters:

- Voice and RAG flows cross multiple services.
- Debugging consent denial, missing audit events, and notification failures requires correlation.
- Evaluation metrics need reliable latency data.

Recommended implementation path:

1. Standardize request/correlation IDs across:
   - API gateway
   - MCP server
   - health-records-service
   - audit-service
   - voice-agent-service
   - notification-service
   - location-service
2. Add OpenTelemetry initialization to every Go service.
3. Pass correlation ID through:
   - HTTP headers
   - gRPC metadata
   - MCP tool metadata
   - RabbitMQ message headers
4. Add metrics:
   - request duration
   - gRPC status counts
   - RabbitMQ publish/consume delay
   - RAG retrieval latency
   - LLM provider latency
   - notification success/failure
5. Keep PHI-safe logging rules enforced through `shared/logging`.

Suggested stack:

- OpenTelemetry Go SDK.
- OTLP collector or Jaeger for local dev.
- Structured logs with zap or slog.
- RabbitMQ headers for correlation.
- Prometheus metrics if the Kubernetes path remains primary.

## Priority Roadmap

### Priority 0: Make the Current Demo Path Reproducible

Goal: one command sequence proves the core product works locally.

Scope:

- Seed demo patient.
- Seed consents.
- Seed FHIR bundle.
- Verify patient login.
- Verify patient records answer returns a citation.
- Verify hospital summary works for a consented patient.
- Verify emergency incident appears after an escalation event.
- Verify nearest-hospital returns a result.

Deliverables:

- executable scripts under `scripts/evaluation`
- clear fixture data under `examples`
- documented expected outputs
- optional CI job that runs deterministic checks without real external providers

### Priority 1: Close Security and Consent Gaps

Goal: make authorization behavior explicit and testable.

Scope:

- role and permission matrix
- refresh/revocation design
- token TTL policy
- consent denial tests
- internal service auth tests
- CORS configuration
- unsafe logging checks

Deliverables:

- `docs/security.md` update with canonical role model
- shared auth tests
- API gateway middleware tests
- MCP gateway denied-path tests

### Priority 2: Stabilize Health Records RAG

Goal: make grounded answers reliable enough for demos and evaluation.

Scope:

- deterministic synthetic QA set
- citation assertions
- consent gating tests
- no-record refusal behavior
- chunk metadata improvements
- retrieval/rerank evaluation

Deliverables:

- `rag-groundedness-check`
- RAG fixtures
- health-records-service tests
- evaluation JSON output

### Priority 3: Package for Contributors

Goal: reduce setup friction.

Scope:

- Docker Compose local demo
- fake/no-send providers
- documented local env
- web README update
- mobile README verification notes

Deliverables:

- `docker-compose.yml`
- `.env.example` alignment
- smoke instructions
- optional Helm chart skeleton later

### Priority 4: Build Provider and Organization Foundations

Goal: make hospital-facing workflows real.

Scope:

- health-provider-service implementation
- organization/facility/staff models
- provider-patient relationships
- staff access policy
- hospital audit and summary authorization

Deliverables:

- provider schema migrations
- service contracts
- API gateway integration
- staff authorization tests

## Suggested Definition of Done for Future Features

For any new healthcare-sensitive feature, require:

- source docs updated
- API or gRPC contract documented
- auth behavior documented
- consent behavior documented
- audit event documented
- PHI logging reviewed
- local smoke path added
- failure mode tested
- provider dependencies abstracted or explicitly justified

## Open Questions

1. Should Docker Compose become the primary contributor path, with Tilt reserved for Kubernetes-focused development?
2. Should the mobile app remain a single Expo app for both patient and hospital staff, or split into role-specific navigation modules first?
3. Should health-provider-service expose gRPC internally only, or also own some direct HTTP endpoints?
4. Which provider abstraction should be tackled first: notification, LLM/embedding, telephony, or translation?
5. Should the evaluation harness be written in Go for contract-level checks, or TypeScript for closer alignment with web/mobile workflows?

## Recommended Next Step

Start with the reproducible demo and evaluation path. It will expose integration gaps faster than another feature pass:

1. Seed one demo patient and FHIR bundle.
2. Grant the needed consent records.
3. Add `patient-portal-smoke`.
4. Add `consent-gating-check`.
5. Add `rag-groundedness-check`.
6. Add `hospital-escalation-smoke`.
7. Add `nearest-hospital-smoke`.

Once these pass locally with fake providers, the codebase will have a much stronger foundation for security hardening, provider abstraction, deployment packaging, and research evaluation.
