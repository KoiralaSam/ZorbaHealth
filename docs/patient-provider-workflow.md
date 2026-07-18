# Patient and Provider Workflow

This workflow is the current UX and security target for the local demo path.

```mermaid
flowchart LR
  Patient["Patient"] -->|"register or log in"| PatientAuth["Patient auth"]
  PatientAuth -->|"open portal"| Portal["Patient portal"]
  Portal -->|"review and grant consent"| Consent["Consent center"]
  Portal -->|"ask health question"| RecordsQA["Health record Q&A"]
  RecordsQA -->|"requires HEALTH_RECORD_ACCESS"| Records["FHIR records and citations"]
  Portal -->|"share emergency location"| Location["Location session"]
  Location -->|"nearest hospital lookup"| Incident["Escalation incident"]

  Provider["Hospital provider"] -->|"log in"| ProviderAuth["Provider auth"]
  ProviderAuth -->|"open dashboard"| Dashboard["Hospital dashboard"]
  Dashboard -->|"review incidents"| Incident
  Dashboard -->|"request summary"| Summary["Patient summary"]

  Consent -->|"authorizes"| Summary
  Summary -->|"audited access"| Audit["Audit trail"]
  Records -->|"audited access"| Audit
  Incident -->|"audited escalation"| Audit
```

## UX Requirements

- Patient workflows must make consent status visible before record Q&A and provider summaries depend on it.
- Provider workflows must separate incident review, patient summary, and patient audit lookup so staff can understand why an action is denied.
- Emergency location sharing must be explicit, session scoped, and revocable by ending the location session.

## Security Requirements

- Patient portal calls require patient tokens.
- Hospital workflows require staff tokens with a hospital identifier.
- Record Q&A and provider summaries require consent-gated access.
- Audit events should exist for successful and denied PHI-sensitive access.
- Incident metadata shown in provider views should stay PHI-minimal by default.
