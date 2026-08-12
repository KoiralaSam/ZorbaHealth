# Seed Data Scripts

This directory now serves as the landing zone for repeatable demo and onboarding seed flows.

## seed-demo.sh

Seeds the local dev cluster (namespace `dev`) with working demo credentials so
authenticated web/mobile flows and `scripts/evaluation/demo-smoke.mjs` pass:

```bash
./scripts/seed-data/seed-demo.sh
```

Creates (idempotently) a demo patient (`+15555550100` / `demo-password`), a
demo hospital with an active staff account (`staff@zorbahealth.local` /
`demo-password`), and an active patient↔hospital consent link. Passwords are
bcrypt-hashed via `hashpw/` at the same cost the auth service uses. Override
credentials with the `DEMO_*` env vars documented in the script header.

## get-otp.sh

The local stack has no real SMS/email delivery, so registration verification
codes never arrive. This helper reads the pending OTP entry from the
in-cluster Redis:

```bash
./scripts/seed-data/get-otp.sh "+15555010299"
# {"token":"<email-verify token>","code":"<sms otp>"}
```

Complete the flow with `POST /api/v1/auth/patient/register/verify-otp`
(`phone_number` + `otp`) followed by `POST /api/v1/auth/patient/register/verify`
(`token`).

## Demo mode intent

The current product roadmap adds a consent-aware demo mode so contributors can:

- log in as a synthetic patient
- exercise the patient web portal
- test consent changes safely
- run voice and escalation demos without production PHI

## Recommended seed set

The first seed bundle should include:

- one demo patient linked to an auth user
- active consent rows for the most common patient-facing flows
- one or more ended calls with summaries in the `calls` table
- a few audit events in `audit.audit_events`
- one synthetic emergency escalation event for the hospital inbox

## Relationship to other assets

- sample FHIR content belongs in `examples/sample-fhir-data`
- evaluation scenarios belong in `scripts/evaluation`
- request examples belong in `examples/sample-requests`

## Suggested script order

1. Seed auth and patient rows.
2. Seed audit consents.
3. Seed recent calls and summaries.
4. Seed emergency escalation events.
5. Print the demo patient phone number and login credentials for local use.
