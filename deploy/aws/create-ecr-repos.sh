#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "${EKS_ENV_FILE:-$ROOT/deploy/aws/eks.env}" 2>/dev/null || true

: "${AWS_REGION:=us-east-1}"
export AWS_REGION AWS_DEFAULT_REGION="$AWS_REGION"
: "${AWS_ACCOUNT_ID:=954976298234}"
: "${ECR_REGISTRY:=${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com}"

SERVICES=(
  patient-service auth-service api-gateway notification-service
  health-records-service analytics-service audit-service location-service
  translation-service interpretation-service
  mcp-server voice-agent-service web mobile
)

for svc in "${SERVICES[@]}"; do
  aws ecr describe-repositories --repository-names "$svc" --region "$AWS_REGION" >/dev/null 2>&1 \
    || aws ecr create-repository --repository-name "$svc" --region "$AWS_REGION" \
         --image-scanning-configuration scanOnPush=true \
         --encryption-configuration encryptionType=AES256
  echo "ok: $ECR_REGISTRY/$svc"
done
