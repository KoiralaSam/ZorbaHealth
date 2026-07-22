#!/usr/bin/env bash
# Reset kind/Tilt Postgres data so it re-initializes from postgres-secret.
# Use when migrate fails with: password authentication failed for user "healthai"
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
NS="${K8S_NAMESPACE:-dev}"

command -v kubectl >/dev/null || { echo "kubectl required" >&2; exit 1; }

echo "[reset-postgres] context=$(kubectl config current-context)"
echo "[reset-postgres] Deleting Deployment/PVC in namespace ${NS}..."
kubectl delete deployment postgres -n "$NS" --ignore-not-found
kubectl delete pvc postgres-data -n "$NS" --ignore-not-found

echo "[reset-postgres] Re-applying development secrets + postgres manifests..."
if [[ -f "$ROOT/deploy/kubernetes/development/secrets.yaml" ]]; then
  kubectl apply -f "$ROOT/deploy/kubernetes/development/secrets.yaml"
else
  echo "Missing secrets.yaml — copy from secrets.example.yaml and fill placeholders." >&2
  exit 1
fi
kubectl apply -k "$ROOT/deploy/kubernetes/development"

echo "[reset-postgres] Waiting for postgres Ready..."
kubectl rollout status deployment/postgres -n "$NS" --timeout=180s
kubectl wait --for=condition=ready pod -l app=postgres -n "$NS" --timeout=180s

echo "[reset-postgres] Done. Re-run Tilt db-migrate (or: ./deploy/tilt/migrate-up.sh)."
