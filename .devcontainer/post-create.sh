#!/usr/bin/env bash
# Ensure tooling exists, then verify Docker / k8s clients.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="${HOME}/.local/bin:/usr/local/bin:${PATH}"

bash "${ROOT}/scripts/codespaces/install-tools.sh"

echo "Verifying Docker / Compose / kubectl…"
docker version >/dev/null
docker compose version
kubectl version --client

echo
echo "Devcontainer ready."
echo
echo "Docker Compose (lighter):"
echo "  ./scripts/codespaces/prepare-env.sh"
echo "  docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.override.codespaces.yml up --build"
echo
echo "Kubernetes via kind + Tilt:"
echo "  ./scripts/codespaces/kind-up.sh"
echo "  ./deploy/tilt/preflight.sh && tilt up"
