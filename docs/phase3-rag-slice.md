# Phase 3 RAG Slice

This document narrows the larger Phase 3 roadmap into a minimal slice that is useful for both product and research work.

## Goal

Deliver one grounded patient-answering path that:

- retrieves patient-specific context
- returns a concise answer
- includes source references
- can be exercised from both the patient portal and the voice agent

## Minimal scope

### Data

- one synthetic patient bundle in `examples/sample-fhir-data`
- a small set of observations, conditions, and medications
- one seed path that loads those records into local development

### Retrieval

- use the existing `AnswerPatientQuestion` flow in `services/health-records-service`
- keep the initial implementation narrow and deterministic
- return citations that point back to source chunks or files

### UX surfaces

- patient web portal question box in `web/src/app/patient/page.tsx`
- voice flow through the MCP server and health records service

## Acceptance criteria

- a seeded demo patient can ask a health question locally
- the answer includes at least one citation
- the response is gated by the patient-scoped token path
- audit events continue to reflect sensitive access

## Recommended follow-on work

After this slice is stable, the next increment should be:

1. FHIR ingestion and normalization improvements
2. pgvector-backed chunk search
3. stronger source-grounding evaluation
4. richer patient-facing explanations of retrieved evidence
