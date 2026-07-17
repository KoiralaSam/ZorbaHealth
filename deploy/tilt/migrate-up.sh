#!/usr/bin/env bash
# Runs golang-migrate against Postgres via Tilt's localhost:5432 port-forward.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
NS="${K8S_NAMESPACE:-dev}"

if [[ -z "${DATABASE_URL:-}" ]]; then
  command -v kubectl >/dev/null 2>&1 || { echo "kubectl required (or set DATABASE_URL)" >&2; exit 1; }
  user="$(kubectl get secret postgres-secret -n "$NS" -o jsonpath='{.data.POSTGRES_USER}' | base64 -d)"
  pass="$(kubectl get secret postgres-secret -n "$NS" -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d)"
  db="$(kubectl get secret postgres-secret -n "$NS" -o jsonpath='{.data.POSTGRES_DB}' | base64 -d)"
  pass_enc="$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=''))" "$pass")"
  export DATABASE_URL="postgres://${user}:${pass_enc}@127.0.0.1:5432/${db}?sslmode=disable"
fi

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
  exit 1
fi

command -v migrate >/dev/null 2>&1 || {
  echo "[db-migrate] Install golang-migrate (e.g. brew install golang-migrate)" >&2
  exit 1
}

make migrate-up
