#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PATH="${HOME}/.local/bin:${PATH}"
# shellcheck disable=SC1091
source "${EKS_ENV_FILE:-$ROOT/deploy/aws/eks.env}" 2>/dev/null || true
export AWS_REGION="${AWS_REGION:-us-east-1}"
export AWS_DEFAULT_REGION="$AWS_REGION"

command -v aws >/dev/null || { echo "aws CLI required" >&2; exit 1; }
command -v kubectl >/dev/null || { echo "kubectl required" >&2; exit 1; }
command -v tilt >/dev/null || { echo "tilt required: https://docs.tilt.dev/install.html" >&2; exit 1; }

aws eks update-kubeconfig --region "$AWS_REGION" --name "${EKS_CLUSTER:-floral-bluegrass-sheepdog}" >/dev/null

if ! aws ecr get-login-password --region "$AWS_REGION" | docker login --username AWS --password-stdin "${ECR_REGISTRY:-954976298234.dkr.ecr.us-east-1.amazonaws.com}" >/dev/null 2>&1; then
  echo "ECR login failed — check IAM (ecr:GetAuthorizationToken)" >&2
  exit 1
fi

kubectl get storageclass gp3 >/dev/null 2>&1 || {
  echo "Applying gp3 StorageClass for EKS Auto Mode..."
  kubectl apply -f "$ROOT/deploy/aws/eks-gp3-storageclass.yaml"
}

if [[ ! -f "$ROOT/deploy/kubernetes/development/secrets.yaml" ]]; then
  echo "Missing deploy/kubernetes/development/secrets.yaml — copy from secrets.example.yaml" >&2
  exit 1
fi

echo "Tilt preflight OK"
