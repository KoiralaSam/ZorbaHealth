# Seed FHIR Data

Loads synthetic FHIR bundles into the development database through `health-records-service` (`IngestFHIRBundle`).

## Prerequisites

- Running stack with `health-records-service` reachable on gRPC port `50054`
- `INTERNAL_SERVICE_SECRET` **exactly equal** to `app-secrets.INTERNAL_SERVICE_SECRET` in the `dev` namespace (same value as `health-records-service` pods)
- An existing internal `patients.id` UUID to attach the bundle to

## Usage

Run from the **repository root** (`ZorbaHealth/`):

```bash
cd /path/to/ZorbaHealth

# Must match the cluster secret (not a hard-coded default):
export INTERNAL_SERVICE_SECRET="$(
  kubectl get secret app-secrets -n dev -o jsonpath='{.data.INTERNAL_SERVICE_SECRET}' | base64 -d
)"

go run ./scripts/seed-fhir-data \
  -patient-id "00000000-0000-0000-0000-000000000001" \
  -bundle examples/sample-fhir-data/demo-patient-bundle.json \
  -source synthetic-demo
```

If `invalid internal token` appears, your export does not match what pods use — re-read from `app-secrets` or set the same string you put in `deploy/kubernetes/development/secrets.yaml` under `app-secrets` → `INTERNAL_SERVICE_SECRET`.

### When `kubectl` fails (broken Homebrew `aws` / pyexpat)

EKS kubeconfig often runs `aws eks get-token`. If you see `pyexpat` / `aws failed with exit code 255`, use the official AWS CLI v2 (see [`deploy/aws/README.md`](../../deploy/aws/README.md)):

```bash
export PATH="$HOME/.local/bin:$PATH"
aws --version   # should work before kubectl
```

Then re-run the `kubectl get secret ...` export above.

### Without `kubectl`

Set the env var to the **same** `INTERNAL_SERVICE_SECRET` string in your local `deploy/kubernetes/development/secrets.yaml` (`app-secrets` section):

```bash
export INTERNAL_SERVICE_SECRET='paste-the-value-from-secrets-yaml'
go run ./scripts/seed-fhir-data ...
```

If `INTERNAL_SERVICE_SECRET is required`, the export is empty (kubectl failed or the key is missing).

If your shell is already in `scripts/seed-fhir-data`, use `.` instead of `./scripts/seed-fhir-data` and paths relative to that folder:

```bash
export INTERNAL_SERVICE_SECRET="$(
  kubectl get secret app-secrets -n dev -o jsonpath='{.data.INTERNAL_SERVICE_SECRET}' | base64 -d
)"
go run . \
  -patient-id "00000000-0000-0000-0000-000000000001" \
  -bundle ../../examples/sample-fhir-data/demo-patient-bundle.json \
  -source synthetic-demo
```

## Idempotency

Re-running the command upserts FHIR resources by `(patient_id, resource_type, resource_id)` and adds new chunk rows. For a clean slate, delete the patient's rows in `records.fhir_resources` and `records.record_chunks`.

## Related docs

- [`examples/sample-fhir-data/README.md`](../examples/sample-fhir-data/README.md)
- [`docs/local-setup.md`](../docs/local-setup.md) (Synthea + HAPI notes)
