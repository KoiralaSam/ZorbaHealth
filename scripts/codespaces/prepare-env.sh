#!/usr/bin/env bash
# Write examples/sample-env/.env.codespaces with Codespaces forwarded HTTPS URLs.
# Prefer .env.docker when present; otherwise seed from .env.example with local defaults.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SAMPLE_DIR="${ROOT}/examples/sample-env"
OUT="${SAMPLE_DIR}/.env.codespaces"
DOCKER_ENV="${SAMPLE_DIR}/.env.docker"
EXAMPLE_ENV="${SAMPLE_DIR}/.env.example"

if [[ -z "${CODESPACE_NAME:-}" ]]; then
  echo "error: CODESPACE_NAME is not set. Run this inside a GitHub Codespace." >&2
  echo "For local Docker Compose, use deploy/docker/docker-compose.override.local.yml instead." >&2
  exit 1
fi

DOMAIN="${GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN:-app.github.dev}"
WEB_URL="https://${CODESPACE_NAME}-3000.${DOMAIN}"
API_URL="https://${CODESPACE_NAME}-8081.${DOMAIN}"
LOC_WS="wss://${CODESPACE_NAME}-8091.${DOMAIN}"

if [[ -f "${DOCKER_ENV}" ]]; then
  SRC="${DOCKER_ENV}"
  echo "Seeding from ${SRC}"
else
  SRC="${EXAMPLE_ENV}"
  echo "Seeding from ${SRC} (no .env.docker found)"
fi

cp "${SRC}" "${OUT}"

# Upsert or replace a KEY=value line in the output file.
upsert() {
  local key="$1"
  local value="$2"
  if grep -q "^${key}=" "${OUT}"; then
    # Use | as sed delimiter to avoid clashes with URLs.
    sed -i.bak "s|^${key}=.*|${key}=${value}|" "${OUT}"
    rm -f "${OUT}.bak"
  else
    printf '%s=%s\n' "${key}" "${value}" >>"${OUT}"
  fi
}

# When seeding from the placeholder example, apply Compose-friendly local defaults.
if [[ "${SRC}" == "${EXAMPLE_ENV}" ]]; then
  upsert "DATABASE_URL" "postgres://healthai:healthai@postgres:5432/healthai?sslmode=disable"
  upsert "POSTGRES_PASSWORD" "healthai"
  upsert "INTERNAL_SERVICE_SECRET" "dev-internal-secret-codespaces-only"
  upsert "AUTH_SERVICE_JWT_SECRET" "local-dev-jwt-secret-change-me-32chars"
  upsert "PATIENT_SERVICE_JWT_SECRET" "local-dev-jwt-secret-change-me-32chars"
  upsert "OPENAI_API_KEY" ""
  upsert "DEEPGRAM_API_KEY" ""
  upsert "ELEVENLABS_API_KEY" ""
  upsert "ELEVENLABS_VOICE_ID" ""
  upsert "LIVEKIT_API_KEY" ""
  upsert "LIVEKIT_API_SECRET" ""
  upsert "LIVEKIT_WS_URL" "ws://localhost:7880"
  upsert "LIVEKIT_URL" "ws://localhost:7880"
  upsert "LIVEKIT_PUBLIC_WS_URL" "ws://localhost:7880"
  upsert "MAILTRAP_API_TOKEN" ""
  upsert "MAILTRAP_FROM_EMAIL" "noreply@localhost"
  upsert "MAILTRAP_FROM_NAME" "ZorbaHealth"
  upsert "MAILTRAP_MIRROR_RECIPIENT" ""
  upsert "VOIPMS_API_USERNAME" ""
  upsert "VOIPMS_API_PASSWORD" ""
  upsert "VOIPMS_DID" ""
  upsert "VOIPMS_API_KEY" ""
  upsert "IP_GEOLOCATION_ENDPOINT_TEMPLATE" "https://ipapi.co/%s/json/"
fi

upsert "API_GATEWAY_ALLOWED_ORIGINS" "${WEB_URL}"
upsert "PUBLIC_WEB_BASE_URL" "${WEB_URL}"
upsert "NEXT_PUBLIC_API_URL" "${API_URL}"
upsert "NEXT_PUBLIC_LOCATION_WS_URL" "${LOC_WS}"

echo "Wrote ${OUT}"
echo "  PUBLIC_WEB_BASE_URL=${WEB_URL}"
echo "  NEXT_PUBLIC_API_URL=${API_URL}"
echo "  NEXT_PUBLIC_LOCATION_WS_URL=${LOC_WS}"
echo
echo "Next:"
echo "  docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.override.codespaces.yml up --build"
echo "  export DATABASE_URL='postgres://healthai:healthai@localhost:5432/healthai?sslmode=disable' && make migrate-up"
echo
echo "In the Ports panel, set visibility to Public for 3000, 8081, and 8091 when using *.${DOMAIN} URLs."
