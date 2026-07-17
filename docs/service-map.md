# Service Map

This document inventories the services currently present in the repository, their entrypoints, interfaces, major dependencies, and current maturity.

## Core application services

### API Gateway

- Path: `services/api-gateway`
- Entrypoint: `services/api-gateway/cmd/api-gateway/main.go`
- Interfaces: HTTP on `API_GATEWAY_HTTP_ADDR` (default `:8081`)
- Primary dependencies: patient service over gRPC
- Current scope: patient login, portal, hospital dashboard routes, and health-staff meeting scheduling HTTP APIs
- Status: Partial

### Auth Service

- Path: `services/auth-service`
- Entrypoint: `services/auth-service/cmd/auth-service/main.go`
- Interfaces: gRPC on `AUTH_SERVICE_GRPC_ADDR`
- Primary dependencies: PostgreSQL, RabbitMQ
- Current scope: login, token verification, registration-related auth support
- Status: Implemented

### Patient Service

- Path: `services/patient-service`
- Entrypoint: `services/patient-service/cmd/patient-service/main.go`
- Interfaces: gRPC on `PATIENT_SERVICE_GRPC_ADDR`
- Primary dependencies: PostgreSQL, Redis, RabbitMQ, auth service
- Current scope: patient registration, verification, portal profile/calls, and `SchedulingService` (pending staff approval, LiveKit room creation, meeting notifications via RabbitMQ)
- Status: Implemented

### Health Records Service

- Path: `services/health-records-service`
- Entrypoint: `services/health-records-service/cmd/health-records-service/main.go`
- Interfaces: gRPC on `MEDICAL_RECORDS_SERVICE_GRPC_ADDR`
- Primary dependencies: PostgreSQL, OpenAI-backed embedding and summarization adapters
- Current scope: health-record retrieval, embeddings, and summarization logic
- Status: Implemented

### MCP Server

- Path: `services/mcp-server`
- Entrypoint: `services/mcp-server/cmd/mcp-server/main.go`
- Interfaces: MCP stdio transport by default; streamable HTTP on `MCP_HTTP_ADDR` when `MCP_TRANSPORT=http`
- Primary dependencies: PostgreSQL, RabbitMQ, audit service, health-records service, translation service, location service, analytics service
- Current scope: tool routing for health-record, translation, location, analytics, escalation, and `schedule_health_staff_meeting` with consent/audit checks
- Status: Implemented

### Audit Service

- Path: `services/audit-service`
- Entrypoint: `services/audit-service/cmd/audit-service/main.go`
- Interfaces: gRPC on `AUDIT_SERVICE_GRPC_ADDR`
- Primary dependencies: PostgreSQL
- Current scope: append-only audit event storage, audit query path, and consent grant/check/revoke/list APIs
- Status: Implemented

### Voice Agent Service

- Path: `services/voice-agent-service`
- Entrypoint: `services/voice-agent-service/src/agent.py`
- Interfaces: outbound LiveKit agent worker; HTTP on `VOICE_AGENT_HTTP_ADDR` for `/webhook/livekit` and `/health`; MCP client to `mcp-server`
- Primary dependencies: LiveKit, MCP server, Deepgram, OpenAI, ElevenLabs
- Current scope: realtime voice session orchestration, LiveKit webhook handling, STT/LLM/TTS, MCP-backed assistant tools, and current safety/escalation trigger logic
- Status: Implemented

### Notification Service

- Path: `services/notification-service`
- Entrypoint: `services/notification-service/cmd/notification-service/main.go`
- Interfaces: HTTP on `HTTP_ADDR`, RabbitMQ consumer
- Primary dependencies: RabbitMQ, Mailtrap, VoIP.ms
- Current scope: patient event notifications, emergency escalation SMS fan-out, and SMS webhook handling
- Status: Implemented

### Location Service

- Path: `services/location-service`
- Entrypoint: `services/location-service/cmd/location-service/main.go`
- Interfaces: gRPC on `LOCATION_SERVICE_GRPC_ADDR`, HTTP WebSocket endpoint on `LOCATION_SERVICE_HTTP_ADDR`
- Primary dependencies: Redis, RabbitMQ, IP geolocation provider
- Current scope: location retrieval, session-linked live location, nearest-hospital lookup
- Status: Partial
- Notes: nearest-hospital support currently uses a noop/stub finder

### Translation Service

- Path: `services/translation-service`
- Entrypoint: `services/translation-service/cmd/translation-service/main.go`
- Interfaces: gRPC on `TRANSLATION_SERVICE_GRPC_ADDR`
- Primary dependencies: Amazon Translate or the legacy local translation-model HTTP backend
- Current scope: provider-switched text translation for MCP and bridged-call interpretation workflows
- Status: Implemented

### Interpretation Service

- Path: `services/interpretation-service`
- Entrypoint: `services/interpretation-service/cmd/interpretation-service/main.go`
- Interfaces: HTTP on `INTERPRETATION_SERVICE_HTTP_ADDR`
- Primary dependencies: Redis, translation service, audit service
- Current scope: bridged-call relay control for per-segment interpretation decisions using session-scoped patient/staff preferences
- Status: Partial

### Analytics Service

- Path: `services/analytics-service`
- Entrypoint: `services/analytics-service/cmd/analytics-service/main.go`
- Interfaces: gRPC on `ANALYTICS_SERVICE_GRPC_ADDR`
- Primary dependencies: PostgreSQL
- Current scope: analytics queries and materialized-view refreshes
- Status: Implemented

### Health Provider Service

- Path: `services/health-provider-service`
- Entrypoint: `services/health-provider-service/cmd/health-provider-service/main.go`
- Interfaces: HTTP on `HEALTH_PROVIDER_SERVICE_HTTP_ADDR`
- Primary dependencies: none wired yet
- Current scope: placeholder for provider and organization APIs
- Status: Future work

## Supporting application modules

### Web application

- Path: `web`
- Stack: Next.js
- Role: patient-facing web frontend for registration, login, and verification flows

### Shared packages

- Path: `shared`
- Role: common Go packages, generated protobuf code, env helpers, DB helpers, auth helpers, messaging helpers

### Proto definitions

- Path: `proto`
- Role: source `.proto` files for service contracts

### Database migrations

- Path: `migrations`
- Role: SQL schema and view migrations

### Infrastructure

- Path: `deploy/docker`, `deploy/kubernetes`
- Role: Dockerfiles and Kubernetes manifests for local and production-like environments

## Dependency summary

- PostgreSQL: auth, patient, health-records, analytics, MCP server
- Redis: patient, location, interpretation
- RabbitMQ: auth, patient, notification, location
- LiveKit and SIP ecosystem: voice agent service, bridged-call interpretation workflow
- AI providers: voice agent service, health-records, translation

## Important current gaps

- No provider-agnostic abstraction layer across all external integrations yet
- No Helm charts or Docker Compose workflow yet
- Health provider functionality is still a stub
