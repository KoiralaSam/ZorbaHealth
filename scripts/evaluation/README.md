# Evaluation Scripts

This directory is reserved for reproducible evaluation tooling tied to the current product roadmap.

## Near-term scenarios

Before the full journal harness is built, contributors should be able to evaluate:

- patient portal login and dashboard load
- consent mutation success and denial paths
- patient record question answering
- hospital summary generation for a consented patient
- emergency escalation visibility in the hospital inbox
- nearest-hospital lookup using both online and fallback paths

## Recommended first scripts

1. `patient-portal-smoke`
   - log in as the demo patient
   - load profile, consents, calls, and audit history

2. `consent-gating-check`
   - revoke `HEALTH_RECORD_ACCESS`
   - verify patient record answer requests fail cleanly
   - grant it again and verify success

3. `hospital-escalation-smoke`
   - log in as hospital staff
   - verify incident list returns at least one seeded escalation
   - verify patient audit lookup respects consent

4. `nearest-hospital-smoke`
   - call the location path with known coordinates
   - verify a hospital name and directions URL are returned

See `docs/q1-journal-evaluation-plan.md` for the broader measurement narrative.

## Demo smoke runner

`demo-smoke.mjs` is the first executable version of the priority-0 demo checks. It uses the API gateway contract and emits JSON that can be saved as local evidence or wired into CI after deterministic seed data is stable.

```bash
node scripts/evaluation/demo-smoke.mjs all
node scripts/evaluation/demo-smoke.mjs patient-portal-smoke
node scripts/evaluation/demo-smoke.mjs consent-gating-check
node scripts/evaluation/demo-smoke.mjs rag-groundedness-check
node scripts/evaluation/demo-smoke.mjs hospital-escalation-smoke
node scripts/evaluation/demo-smoke.mjs nearest-hospital-smoke
```

Supported environment variables:

- `API_GATEWAY_URL` defaults to `http://localhost:8081`
- `DEMO_PATIENT_PHONE`, `DEMO_PATIENT_EMAIL`, `DEMO_PATIENT_PASSWORD`
- `DEMO_HOSPITAL_EMAIL`, `DEMO_HOSPITAL_PASSWORD`
- `DEMO_PATIENT_ID` for hospital audit checks when the incident list is empty
- `DEMO_HEALTH_QUESTION` for the RAG citation check
- `LOCATION_NEAREST_HOSPITAL_URL` for an HTTP wrapper around location-service `FindNearestHospital`

Security notes:

- The runner does not print bearer tokens.
- The JSON output stores status, latency, assertion names, and non-secret IDs only.
- `consent-gating-check` restores `HEALTH_RECORD_ACCESS` after testing the denial path.

## Full artifact run

For the current reproducible harness entrypoint:

```bash
./scripts/evaluation/run_all.sh
```

That command writes an artifact directory containing:

- `demo-smoke.json`
- `summary.json`
- `report-template.md`

Additional planned inputs now live under:

- `scripts/evaluation/load/api-gateway.js`
- `scripts/evaluation/datasets/rag-qrels.jsonl`
- `scripts/evaluation/datasets/translation-cases.jsonl`
