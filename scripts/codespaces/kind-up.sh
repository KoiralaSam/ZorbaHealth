#!/usr/bin/env bash
# Create (or reuse) a kind cluster for Tilt in Codespaces / DinD.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLUSTER_NAME="${KIND_CLUSTER_NAME:-zorba}"
CONFIG="${ROOT}/scripts/codespaces/kind-config.yaml"
SECRETS="${ROOT}/deploy/kubernetes/development/secrets.yaml"
SECRETS_EXAMPLE="${ROOT}/deploy/kubernetes/development/secrets.example.yaml"

export PATH="${HOME}/.local/bin:${PATH}"

command -v docker >/dev/null || { echo "docker required" >&2; exit 1; }
command -v kind >/dev/null || { echo "kind required (installed by .devcontainer/post-create.sh)" >&2; exit 1; }
command -v kubectl >/dev/null || { echo "kubectl required" >&2; exit 1; }

if ! docker info >/dev/null 2>&1; then
  echo "Docker daemon is not reachable. Wait for DinD to finish starting, then retry." >&2
  exit 1
fi

if kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  echo "kind cluster '${CLUSTER_NAME}' already exists"
else
  echo "Creating kind cluster '${CLUSTER_NAME}'…"
  kind create cluster --name "${CLUSTER_NAME}" --config "${CONFIG}" --wait 120s
fi

kubectl config use-context "kind-${CLUSTER_NAME}" >/dev/null
kubectl cluster-info

kubectl create namespace dev --dry-run=client -o yaml | kubectl apply -f -

if [[ ! -f "${SECRETS}" ]]; then
  cp "${SECRETS_EXAMPLE}" "${SECRETS}"
  echo
  echo "Created ${SECRETS} from the example template."
  echo "Replace every REPLACE_* placeholder before running tilt up."
  echo
fi

echo "kind ready (context: kind-${CLUSTER_NAME})"
echo "Next:"
echo "  1. Edit deploy/kubernetes/development/secrets.yaml (if still placeholders)"
echo "  2. ./deploy/tilt/preflight.sh"
echo "  3. tilt up"
echo
echo "Note: prefer 8-core / 32GB+ storage Codespaces for the full Tilt stack."
echo "Compose remains lighter: see docs/local-setup.md Option B."
