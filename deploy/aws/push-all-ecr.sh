#!/usr/bin/env bash
# Build and push all ZorbaHealth service images to ECR (:latest).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "${EKS_ENV_FILE:-$ROOT/deploy/aws/eks.env}" 2>/dev/null || true
: "${AWS_REGION:=us-east-1}"
: "${ECR_REGISTRY:=954976298234.dkr.ecr.us-east-1.amazonaws.com}"
export AWS_REGION AWS_DEFAULT_REGION="$AWS_REGION"

aws ecr get-login-password --region "$AWS_REGION" | \
  docker login --username AWS --password-stdin "$ECR_REGISTRY"

# EKS Auto Mode mixes amd64 + arm64 nodes; Mac builds are arm64-only unless we cross-build.
if ! docker buildx inspect zorba-ecr >/dev/null 2>&1; then
  docker buildx create --name zorba-ecr --use --driver docker-container
else
  docker buildx use zorba-ecr
fi
BUILD_PLATFORM="${BUILD_PLATFORM:-linux/amd64,linux/arm64}"

build_push() {
  local name="$1"
  local dockerfile="$2"
  local ctx="${3:-.}"
  echo "=== $name ($BUILD_PLATFORM) ==="
  docker buildx build \
    --platform "$BUILD_PLATFORM" \
    -f "$dockerfile" \
    -t "$ECR_REGISTRY/$name:latest" \
    --push \
    "$ctx"
}

build_push patient-service deploy/docker/development/patient-service.Dockerfile
build_push auth-service deploy/docker/development/auth-service.Dockerfile
build_push api-gateway deploy/docker/development/api-gateway.Dockerfile .
build_push notification-service deploy/docker/development/notification-service.Dockerfile
build_push health-records-service deploy/docker/development/health-records-service.Dockerfile
build_push analytics-service deploy/docker/development/analytics-service.Dockerfile
build_push audit-service deploy/docker/development/audit-service.Dockerfile
build_push location-service deploy/docker/development/location-service.Dockerfile
build_push translation-service deploy/docker/development/translation-service.Dockerfile
build_push interpretation-service deploy/docker/development/interpretation-service.Dockerfile
build_push mcp-server deploy/docker/development/mcp-server.Dockerfile
build_push voice-agent-service services/voice-agent-service/Dockerfile services/voice-agent-service
build_push web deploy/docker/development/web.Dockerfile
build_push mobile deploy/docker/development/mobile.Dockerfile

echo "All images pushed to $ECR_REGISTRY"
