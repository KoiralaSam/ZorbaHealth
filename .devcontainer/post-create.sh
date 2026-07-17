#!/usr/bin/env bash
set -euo pipefail

echo "Installing golang-migrate (postgres)…"
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1

echo "Verifying Docker / Compose…"
docker version >/dev/null
docker compose version

echo "Devcontainer tooling ready."
echo "Start the stack with:"
echo "  ./scripts/codespaces/prepare-env.sh"
echo "  docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.override.codespaces.yml up --build"
