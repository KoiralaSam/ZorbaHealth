# Seed Data Scripts

This directory now serves as the landing zone for repeatable demo and onboarding seed flows.

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
