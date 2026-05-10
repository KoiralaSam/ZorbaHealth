# Compliance

Zorba Health is intended to evolve into a privacy-aware and clinically safer healthcare AI platform. This document captures the current baseline and the compliance-oriented architecture that later phases will implement.

## Current baseline

- PostgreSQL and Redis-backed services exist today
- analytics functionality exists today through `services/analytics-service`
- health-record access and summarization paths already exist
- there is a legacy `mcp_audit_log` table in the current schema

## Gaps to close

- no dedicated `audit-service` yet
- no append-only audit service boundary yet
- no formal consent-management module yet
- analytics and compliance concerns are not fully separated
- no repository-wide retention policy documentation yet

## Intended compliance model

### Audit versus analytics

- Audit is the compliance source of truth
- Analytics is for aggregate operational insight
- Analytics should avoid raw patient content, raw transcripts, and direct PHI exposure

### Consent

Future phases will add patient consent checks before:

- health-record retrieval
- AI summarization
- location access
- notifications where consent applies
- third-party model processing

### Retention expectations

The architecture direction is:

- temporary Redis-backed session state expires automatically
- location sessions expire after the related call or session
- transcripts should default to disabled unless intentionally enabled
- audit data should follow explicit retention policy rather than ad hoc deletion

## Contributor expectations

- document any feature that introduces PHI handling
- note whether new features belong to analytics, audit, or both
- avoid adding features that blur compliance and product analytics without documenting the rationale
