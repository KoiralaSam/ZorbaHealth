# Zorba Health Web

Next.js App Router frontend for the patient and hospital-provider workflows.

## Local Setup

```bash
cp .env.example .env.local
npm install
npm run dev
```

The app defaults to:

- web: `http://localhost:3000`
- API gateway: `http://localhost:8081`
- location WebSocket: `ws://localhost:8091`

The API gateway must allow the web origin:

```bash
API_GATEWAY_ALLOWED_ORIGINS=http://localhost:3000
```

For preview or production deployments, set `API_GATEWAY_ALLOWED_ORIGINS` to the exact comma-separated browser origins that should be trusted. Do not use `*` with credentialed healthcare routes.

## Environment

- `NEXT_PUBLIC_API_URL`: browser-facing API gateway URL
- `NEXT_PUBLIC_LOCATION_WS_URL`: location-service WebSocket URL for live emergency location sessions
- `NEXT_PUBLIC_LOCATION_HTTP_URL`: location-service HTTP URL for approximate IP fallback

## Patient Workflow

- `/register/patient`: start patient account registration
- `/verify-email`: finish email verification from registration links
- `/login/patient`: patient login
- `/patient`: patient portal for profile, consent center, record Q&A, call summaries, audit trail, and emergency location sharing

Patient UX expectations:

- Raw health records and bearer tokens must not be persisted in browser storage beyond the current implementation needs.
- Consent controls should make grant/revoke state clear before record Q&A or provider summaries depend on it.
- Emergency location sharing should stay explicit and session scoped.

## Hospital Provider Workflow

- `/register/hospital`: start hospital registration
- `/login/hospital`: hospital staff login
- `/login/hospital_staff`: explicit hospital staff login alias
- `/hospital/dashboard`: hospital staff dashboard for patient summaries, meeting requests, emergency incidents, and patient audit lookup
- `/hospital_staff/dashboard`: explicit hospital staff dashboard alias

Provider UX expectations:

- Staff-only screens must require hospital staff tokens.
- Patient summary and audit lookup should fail closed when consent or authorization is missing.
- Incident metadata displayed to staff should stay PHI-minimal unless a specific consented workflow needs more detail.

## Verification

After the gateway and backing services are running:

```bash
node ../scripts/evaluation/demo-smoke.mjs patient-portal-smoke
node ../scripts/evaluation/demo-smoke.mjs hospital-escalation-smoke
```

Use the full runner from the repository root when seed data is available:

```bash
node scripts/evaluation/demo-smoke.mjs all
```
