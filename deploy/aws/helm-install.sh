#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "${EKS_ENV_FILE:-$ROOT/deploy/aws/eks.env}" 2>/dev/null || true

: "${AWS_REGION:=us-east-1}"
export AWS_REGION AWS_DEFAULT_REGION="$AWS_REGION"
: "${EKS_CLUSTER:=floral-bluegrass-sheepdog}"
: "${K8S_NAMESPACE:=dev}"

INFRA_ONLY=false
HELM_WAIT=true
for arg in "$@"; do
  case "$arg" in
    --infra-only) INFRA_ONLY=true ;;
    --no-wait) HELM_WAIT=false ;;
  esac
done

aws eks update-kubeconfig --region "$AWS_REGION" --name "$EKS_CLUSTER"

helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>/dev/null || true
helm dependency build "$ROOT/deploy/helm/"

kubectl create namespace "$K8S_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
if [[ -f "$ROOT/deploy/kubernetes/development/secrets.yaml" ]]; then
  kubectl apply -f "$ROOT/deploy/kubernetes/development/secrets.yaml" -n "$K8S_NAMESPACE"
else
  echo "ERROR: missing deploy/kubernetes/development/secrets.yaml (copy from secrets.example.yaml)" >&2
  exit 1
fi

if [[ "$INFRA_ONLY" == true ]]; then
  echo "Installing infra subchart only (zorba-infra)..."
  HELM_ARGS=(
    upgrade --install zorba-infra "$ROOT/deploy/helm/charts/infra"
    --namespace "$K8S_NAMESPACE"
    -f "$ROOT/deploy/helm/values/eks-infra-only.yaml"
    --create-namespace
  )
else
  HELM_ARGS=(
    upgrade --install zorbahealth "$ROOT/deploy/helm/"
    --namespace "$K8S_NAMESPACE"
    -f "$ROOT/deploy/helm/values/eks-dev.yaml"
    --create-namespace
  )
fi

if [[ "$HELM_WAIT" == true ]]; then
  HELM_ARGS+=(--wait --timeout 20m)
else
  echo "Installing without --wait; diagnose with: kubectl get pods,events -n $K8S_NAMESPACE"
fi

helm "${HELM_ARGS[@]}"
