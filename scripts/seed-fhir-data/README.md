# Seed FHIR Data

Loads synthetic FHIR bundles into the development database through `health-records-service` (`IngestFHIRBundle`).

## Prerequisites

- Running stack with `health-records-service` reachable on gRPC port `50054`
- `INTERNAL_SERVICE_SECRET` matching cluster secrets (used by `shared/grpcclient`)
- An existing internal `patients.id` UUID to attach the bundle to

## Usage

```bash
export INTERNAL_SERVICE_SECRET=dev-internal-secret
go run ./scripts/seed-fhir-data \
  -patient-id "00000000-0000-0000-0000-000000000001" \
  -bundle examples/sample-fhir-data/demo-patient-bundle.json \
  -source synthetic-demo
```

## Idempotency

Re-running the command upserts FHIR resources by `(patient_id, resource_type, resource_id)` and adds new chunk rows. For a clean slate, delete the patient's rows in `records.fhir_resources` and `records.record_chunks`.

## Related docs

- [`examples/sample-fhir-data/README.md`](../examples/sample-fhir-data/README.md)
- [`docs/local-setup.md`](../docs/local-setup.md) (Synthea + HAPI notes)
