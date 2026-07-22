#!/usr/bin/env bash
# Runs golang-migrate against Postgres via Tilt's localhost:5432 port-forward.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
NS="${K8S_NAMESPACE:-dev}"

build_database_url_from_secret() {
  command -v kubectl >/dev/null 2>&1 || { echo "kubectl required (or set DATABASE_URL)" >&2; exit 1; }
  local user pass db pass_enc
  user="$(kubectl get secret postgres-secret -n "$NS" -o jsonpath='{.data.POSTGRES_USER}' | base64 -d)"
  pass="$(kubectl get secret postgres-secret -n "$NS" -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d)"
  db="$(kubectl get secret postgres-secret -n "$NS" -o jsonpath='{.data.POSTGRES_DB}' | base64 -d)"
  if [[ -z "$user" || -z "$pass" || -z "$db" ]]; then
    echo "[db-migrate] postgres-secret missing POSTGRES_USER/PASSWORD/DB in namespace ${NS}" >&2
    exit 1
  fi
  if [[ "$pass" == *"REPLACE"* ]]; then
    echo "[db-migrate] postgres-secret still has a REPLACE_* placeholder password." >&2
    echo "  Edit deploy/kubernetes/development/secrets.yaml, then: kubectl apply -f deploy/kubernetes/development/secrets.yaml" >&2
    exit 1
  fi
  pass_enc="$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=''))" "$pass")"
  export DATABASE_URL="postgres://${user}:${pass_enc}@127.0.0.1:5432/${db}?sslmode=disable"
}

# Always rebuild from the live cluster secret so stale shell/Tilt env cannot drift.
build_database_url_from_secret

echo "[db-migrate] Waiting for Postgres on 127.0.0.1:5432 (Tilt port-forward)..."
ready=0
for _ in $(seq 1 180); do
  if python3 -c "import socket; s=socket.socket(); s.settimeout(1); s.connect(('127.0.0.1',5432))" 2>/dev/null; then
    ready=1
    break
  fi
  sleep 1
done
if [[ "$ready" -ne 1 ]]; then
  echo "[db-migrate] Postgres not reachable on localhost:5432" >&2
  echo "  Ensure Tilt port-forward for postgres is up (resource 'postgres')." >&2
  exit 1
fi

command -v migrate >/dev/null 2>&1 || {
  echo "[db-migrate] Install golang-migrate (e.g. brew install golang-migrate / ./scripts/codespaces/install-tools.sh)" >&2
  exit 1
}

set +e
make migrate-up
status=$?
set -e
if [[ "$status" -ne 0 ]]; then
  echo >&2
  echo "[db-migrate] Migration failed (often a Postgres password mismatch)." >&2
  echo "  Postgres only reads POSTGRES_PASSWORD on first init of the PVC." >&2
  echo "  If you changed secrets.yaml after the volume was created, reset data:" >&2
  echo "    ./deploy/tilt/reset-postgres.sh" >&2
  echo "  Then trigger Tilt resource db-migrate again." >&2
  exit "$status"
fi
