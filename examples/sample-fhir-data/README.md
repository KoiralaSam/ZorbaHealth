# Sample FHIR Data

Synthetic, non-production bundles for local development and evaluation.

## Included bundle

| File | Purpose |
| --- | --- |
| `demo-patient-bundle.json` | Demo patient **Alex Demo** — asthma condition, peak flow observation, albuterol medication, penicillin allergy |

Align the target `patient-id` in [`scripts/seed-fhir-data/README.md`](../scripts/seed-fhir-data/README.md) with your demo auth/patient seed.

## Generate more data

1. Install [Synthea](https://github.com/synthetichealth/synthea) and export an R4 bundle for a single synthetic patient.
2. Drop the bundle JSON in this directory.
3. Run `go run ./scripts/seed-fhir-data -patient-id <uuid> -bundle examples/sample-fhir-data/<your-bundle>.json`.

## Validate locally

```bash
go test ./services/health-records-service/internal/fhir/... -count=1
```

Cross-check resource shapes with a local [HAPI FHIR](https://hapifhir.io/) instance (see [`docs/local-setup.md`](../docs/local-setup.md)).
