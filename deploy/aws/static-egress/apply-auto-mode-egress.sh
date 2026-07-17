#!/usr/bin/env bash
# EKS Auto Mode path: NodeClass + NodePool on NAT private subnet (avoid classic MNG — no VPC CNI here).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${EKS_ENV_FILE:-$ROOT/deploy/aws/eks.env}" 2>/dev/null || true
# shellcheck disable=SC1091
source "$DIR/egress.env"

: "${AWS_REGION:=us-east-1}"
: "${EKS_CLUSTER:=floral-bluegrass-sheepdog}"
: "${STATIC_EGRESS_NODEGROUP_NAME:=static-egress}"
export AWS_REGION AWS_DEFAULT_REGION="$AWS_REGION"

log() { printf '[static-egress] %s\n' "$*"; }

if [[ -z "${STATIC_EGRESS_PRIVATE_SUBNET_ID:-}" ]]; then
  echo "ERROR: run ./provision-network.sh first." >&2
  exit 1
fi

# Patch NodeClass subnet if egress.env differs from the committed default.
nc="$DIR/k8s-nodeclass-static-egress.yaml"
if grep -q 'subnet-0523d6c3d4836a1c6' "$nc" && [[ "$STATIC_EGRESS_PRIVATE_SUBNET_ID" != "subnet-0523d6c3d4836a1c6" ]]; then
  sed "s/subnet-0523d6c3d4836a1c6/${STATIC_EGRESS_PRIVATE_SUBNET_ID}/" "$nc" | kubectl apply -f -
else
  kubectl apply -f "$nc"
fi
kubectl apply -f "$DIR/k8s-nodepool-static-egress.yaml"

status="$(aws eks describe-nodegroup --cluster-name "$EKS_CLUSTER" --nodegroup-name "$STATIC_EGRESS_NODEGROUP_NAME" \
  --region "$AWS_REGION" --query 'nodegroup.status' --output text 2>/dev/null || echo "MISSING")"
if [[ "$status" != "MISSING" ]]; then
  log "Removing classic managed node group $STATIC_EGRESS_NODEGROUP_NAME (status=$status) — not compatible with Auto Mode CNI"
  aws eks delete-nodegroup --cluster-name "$EKS_CLUSTER" --nodegroup-name "$STATIC_EGRESS_NODEGROUP_NAME" --region "$AWS_REGION" \
    >/dev/null 2>&1 || true
  log "Waiting for managed node group delete (often 5–15 min, no AWS progress lines)..."
  while true; do
    st="$(aws eks describe-nodegroup --cluster-name "$EKS_CLUSTER" --nodegroup-name "$STATIC_EGRESS_NODEGROUP_NAME" \
      --region "$AWS_REGION" --query 'nodegroup.status' --output text 2>/dev/null || echo "MISSING")"
    [[ "$st" == "MISSING" ]] && break
    log "managed node group status=$st ($(date +%H:%M:%S))"
    [[ "$st" == "DELETING" ]] || break
    sleep 30
  done
  log "Managed node group removed (or delete still finishing in AWS)"
fi

log "Trigger node provisioning (notification-service nodeSelector + taint):"
kubectl rollout restart deployment/notification-service -n dev 2>/dev/null || true

log "Watch:"
log "  kubectl get nodeclass,nodepool static-egress"
log "  kubectl get nodes -l zorbahealth.io/egress=static -w"
log "  kubectl get pods -n dev -l app=notification-service -o wide"
log "VoIP.ms allowlist: NAT_EIP=${NAT_EIP:-unknown}"
