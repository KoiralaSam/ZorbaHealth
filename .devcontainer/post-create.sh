#!/usr/bin/env bash
# Lightweight checks after the GHCR base image + features are applied.
set -euo pipefail

export PATH="${HOME}/.local/bin:/usr/local/bin:${PATH}"

echo "Verifying tooling…"
docker version >/dev/null
docker compose version
kubectl version --client
kind version
tilt version
migrate -version

echo
echo "Devcontainer ready (image: ghcr.io/koiralasam/zorbahealth-devcontainer)."
echo
echo "Docker Compose (lighter):"
echo "  ./scripts/codespaces/prepare-env.sh"
echo "  docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.override.codespaces.yml up --build"
echo
echo "Kubernetes via kind + Tilt (heavier; prefer 8-core / 32GB+ storage):"
echo "  ./scripts/codespaces/kind-up.sh"
echo "  ./deploy/tilt/preflight.sh && tilt up"
