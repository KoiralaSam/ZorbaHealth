#!/usr/bin/env bash
set -euo pipefail

export PATH="${HOME}/.local/bin:${PATH}"

echo "Installing golang-migrate (postgres)…"
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1

echo "Installing kind…"
KIND_VERSION="${KIND_VERSION:-v0.27.0}"
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64) KIND_ARCH=amd64 ;;
  aarch64|arm64) KIND_ARCH=arm64 ;;
  *) echo "unsupported arch: ${ARCH}" >&2; exit 1 ;;
esac
curl -fsSL -o /tmp/kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-${KIND_ARCH}"
sudo install -m 0755 /tmp/kind /usr/local/bin/kind
rm -f /tmp/kind

echo "Installing Tilt…"
curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash

echo "Verifying tooling…"
docker version >/dev/null
docker compose version
kubectl version --client
kind version
tilt version
migrate -version

echo
echo "Devcontainer tooling ready."
echo
echo "Docker Compose (lighter):"
echo "  ./scripts/codespaces/prepare-env.sh"
echo "  docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.override.codespaces.yml up --build"
echo
echo "Kubernetes via kind + Tilt (heavier; prefer 8-core / 32GB+ storage):"
echo "  ./scripts/codespaces/kind-up.sh"
echo "  ./deploy/tilt/preflight.sh && tilt up"
