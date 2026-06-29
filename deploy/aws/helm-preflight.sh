#!/usr/bin/env bash
# Run before first helm install on EKS. Surfaces the usual blockers with fixes.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "${EKS_ENV_FILE:-$ROOT/deploy/aws/eks.env}" 2>/dev/null || true

: "${AWS_REGION:=us-east-1}"
export AWS_REGION AWS_DEFAULT_REGION="$AWS_REGION"
: "${AWS_ACCOUNT_ID:=954976298234}"
: "${EKS_CLUSTER:=floral-bluegrass-sheepdog}"
: "${ECR_REGISTRY:=${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com}"
: "${K8S_NAMESPACE:=dev}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

fail=0
warn() { echo -e "${YELLOW}WARN${NC} $*"; }
ok() { echo -e "${GREEN}OK${NC} $*"; }
bad() { echo -e "${RED}FAIL${NC} $*"; fail=1; }

echo "=== ZorbaHealth EKS Helm preflight ==="
echo "Cluster: $EKS_CLUSTER ($AWS_REGION)  Namespace: $K8S_NAMESPACE"
echo

for cmd in aws kubectl helm; do
  if command -v "$cmd" >/dev/null 2>&1; then ok "$cmd found"; else bad "missing $cmd (install AWS CLI / kubectl / helm)"; fi
done
[[ "$fail" -eq 1 ]] && exit 1

echo
echo "--- kubeconfig ---"
aws eks update-kubeconfig --region "$AWS_REGION" --name "$EKS_CLUSTER" >/dev/null
CTX="$(kubectl config current-context 2>/dev/null || true)"
ok "current context: $CTX"
if [[ "$CTX" != *"$EKS_CLUSTER"* && "$CTX" != *"eks"* ]]; then
  warn "context may not be EKS; expected name containing '$EKS_CLUSTER'"
fi
kubectl cluster-info 2>/dev/null | head -1 || bad "cannot reach API server (credentials / network)"

echo
echo "--- nodes ---"
if ! kubectl get nodes -o wide 2>/dev/null | tail -n +2 | grep -q .; then
  bad "no Ready nodes — wait for EKS compute or check Auto Mode / node group"
else
  kubectl get nodes -o custom-columns=NAME:.metadata.name,STATUS:.status.conditions[-1].type,CPU:.status.capacity.cpu,MEM:.status.capacity.memory 2>/dev/null
fi

echo
echo "--- storage classes (Postgres/RabbitMQ need gp3 or change values) ---"
kubectl get storageclass -o wide 2>/dev/null || true
if kubectl get storageclass gp3 >/dev/null 2>&1; then
  ok "StorageClass gp3 exists"
elif kubectl get storageclass gp2 >/dev/null 2>&1; then
  ok "StorageClass gp2 exists (eks-dev.yaml defaults to gp2 for this cluster)"
else
  bad "No gp2/gp3 StorageClass — Postgres/RabbitMQ PVCs will stay Pending"
  echo "  Fix: install EBS CSI and create gp3, OR set infra.postgres.storageClass / rabbitmq to an existing class:"
  echo "    helm upgrade ... --set infra.postgres.storageClass=gp2 --set infra.rabbitmq.storageClass=gp2"
fi

echo
echo "--- namespace & secrets (apply deploy/kubernetes/development/secrets.yaml first) ---"
kubectl get namespace "$K8S_NAMESPACE" >/dev/null 2>&1 && ok "namespace $K8S_NAMESPACE" || warn "namespace $K8S_NAMESPACE missing (helm --create-namespace is fine)"
for sec in postgres-secret rabbitmq-credentials app-secrets; do
  if kubectl get secret "$sec" -n "$K8S_NAMESPACE" >/dev/null 2>&1; then
    ok "secret/$sec"
  else
    bad "secret/$sec missing in $K8S_NAMESPACE"
    echo "  Fix: cp deploy/kubernetes/development/secrets.example.yaml secrets.yaml, edit, then:"
    echo "    kubectl apply -f deploy/kubernetes/development/secrets.yaml -n $K8S_NAMESPACE"
  fi
done

echo
echo "--- ECR images (empty repos => ImagePullBackOff; helm --wait will timeout) ---"
SERVICES=(
  patient-service auth-service api-gateway notification-service
  health-records-service analytics-service audit-service location-service
  translation-service interpretation-service
  mcp-server voice-agent-service web mobile
)
missing_ecr=0
for svc in "${SERVICES[@]}"; do
  if ! aws ecr describe-repositories --repository-names "$svc" --region "$AWS_REGION" >/dev/null 2>&1; then
    warn "ECR repo missing: $svc (run deploy/aws/create-ecr-repos.sh)"
    missing_ecr=1
    continue
  fi
  count="$(aws ecr list-images --repository-name "$svc" --region "$AWS_REGION" --query 'length(imageIds)' --output text 2>/dev/null || echo 0)"
  if [[ "$count" == "0" || "$count" == "None" ]]; then
    warn "ECR repo empty: $svc — push at least :latest before full stack"
    missing_ecr=1
  else
    ok "ECR $svc ($count image(s))"
  fi
done
if [[ "$missing_ecr" -eq 1 ]]; then
  echo
  echo "  Build/push one service example:"
  echo "    aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_REGISTRY"
  echo "    docker build -f deploy/docker/development/patient-service.Dockerfile -t $ECR_REGISTRY/patient-service:latest ."
  echo "    docker push $ECR_REGISTRY/patient-service:latest"
  echo "  Or install infra only first (see deploy/aws/README.md#staged-install)."
fi

echo
echo "--- helm render ---"
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>/dev/null || true
helm dependency build "$ROOT/deploy/helm/" >/dev/null
if helm template zorbahealth "$ROOT/deploy/helm/" -f "$ROOT/deploy/helm/values/eks-dev.yaml" -n "$K8S_NAMESPACE" >/dev/null 2>&1; then
  ok "helm template succeeds"
else
  bad "helm template failed — chart syntax/values error"
fi

echo
if [[ "$fail" -eq 1 ]]; then
  echo -e "${RED}Preflight failed — fix FAIL items above before helm install.${NC}"
  exit 1
fi
echo -e "${GREEN}Preflight passed (WARN items may still cause runtime failures).${NC}"
echo "Next: ./deploy/aws/helm-install.sh   or staged: ./deploy/aws/helm-install.sh --infra-only"
