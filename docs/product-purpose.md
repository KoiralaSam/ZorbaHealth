# Product Purpose

Zorba Health exists to make healthcare support available through the most accessible interface most patients already have: the phone.

## Core product promise

The platform combines:

- voice-first access for patients who may not want to install or learn a new app
- controlled backend tool use through the MCP gateway
- explicit consent and audit controls for sensitive healthcare actions
- a provider-facing surface for staff who need summaries, escalation visibility, and operational context

## Primary user journeys

### Patient journey

1. Register or log in.
2. Grant consent for the capabilities they want to use.
3. Start a voice session by phone.
4. Ask questions, receive summaries, and get follow-up communication.
5. Review recent summaries and consent state from the web portal or future mobile app.

### Hospital journey

1. Log in as staff.
2. Retrieve a patient summary after consent is verified.
3. Review emergency escalations and the audit trail around sensitive interactions.
4. Follow up outside the call using notifications or provider workflows.

## Why the architecture is shaped this way

- `services/voice-agent-service` handles realtime voice sessions because telephony and LLM turn-taking require a dedicated runtime.
- `services/mcp-server` acts as a controlled gateway so the conversational agent does not directly reach backend systems.
- `services/audit-service` and `docs/compliance.md` define the compliance boundary for consent and append-only auditing.
- `services/health-records-service` owns patient-context retrieval and is the natural place for the Phase 3 FHIR and RAG work.
- `services/api-gateway` is the correct place to expose web and mobile surfaces without coupling those clients directly to internal gRPC services.

## Current product gap

The repository already had strong backend and research foundations, but the user-facing product was still thinner than the architecture implied:

- patient flows largely stopped at registration and login
- hospital UX was limited to a single summary form
- nearest-hospital lookup was still a noop
- mobile access had no implementation-ready contract

The current feature roadmap work closes those gaps with:

- a real patient home in `web/src/app/patient/page.tsx`
- expanded patient and hospital APIs in `services/api-gateway`
- emergency incident and audit visibility for hospital staff
- a real nearest-hospital adapter in `services/location-service`

## Research alignment

The product still supports the journal narrative described in `docs/q1-journal-evaluation-plan.md`:

- voice-based healthcare assistance
- controlled MCP tool access
- grounded patient context retrieval
- explicit safety, escalation, and consent behavior
- reproducible demo and evaluation assets
