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
