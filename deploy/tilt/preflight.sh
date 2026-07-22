#!/usr/bin/env bash
# Local Kubernetes preflight for Tilt (no cloud provider required).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PATH="${HOME}/.local/bin:${PATH}"

command -v docker >/dev/null || { echo "docker required" >&2; exit 1; }
command -v kubectl >/dev/null || { echo "kubectl required" >&2; exit 1; }
command -v tilt >/dev/null || { echo "tilt required: https://docs.tilt.dev/install.html" >&2; exit 1; }

if ! docker info >/dev/null 2>&1; then
  echo "Docker daemon is not reachable" >&2
  exit 1
fi

if ! kubectl cluster-info >/dev/null 2>&1; then
  echo "kubectl cannot reach a cluster." >&2
  echo "  Local: start Docker Desktop Kubernetes, Minikube, or kind." >&2
  exit 1
fi

if [[ ! -f "$ROOT/deploy/kubernetes/development/secrets.yaml" ]]; then
  echo "Missing deploy/kubernetes/development/secrets.yaml — copy from secrets.example.yaml" >&2
  exit 1
fi

echo "Tilt preflight OK (context: $(kubectl config current-context))"
