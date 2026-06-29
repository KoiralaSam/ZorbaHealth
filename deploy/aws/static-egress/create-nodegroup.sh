#!/usr/bin/env bash
# Managed node group in the static-egress private subnet only (AWS API — no eksctl VPC rules).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${EKS_ENV_FILE:-$ROOT/deploy/aws/eks.env}" 2>/dev/null || true
# shellcheck disable=SC1091
source "$DIR/egress.env"

: "${AWS_REGION:=us-east-1}"
: "${AWS_ACCOUNT_ID:=954976298234}"
: "${EKS_CLUSTER:=floral-bluegrass-sheepdog}"
: "${STATIC_EGRESS_NODE_ROLE_NAME:=zorbahealth-static-egress-node}"
: "${STATIC_EGRESS_NODEGROUP_NAME:=static-egress}"
export AWS_REGION AWS_DEFAULT_REGION="$AWS_REGION"

NODE_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/${STATIC_EGRESS_NODE_ROLE_NAME}"

log() { printf '[static-egress] %s\n' "$*"; }

wait_nodegroup_active() {
  log "Waiting for ACTIVE (often 5–15 min; no output until AWS reports progress)..."
  local deadline=$((SECONDS + 1800))
  while (( SECONDS < deadline )); do
    local st health
    st="$(ng_status)"
    case "$st" in
      ACTIVE)
        log "Node group is ACTIVE"
        return 0
        ;;
      CREATE_FAILED|DELETE_FAILED)
        log "Node group failed: $st"
        print_failure
        return 1
        ;;
      MISSING)
        log "Node group disappeared while waiting (status=MISSING)"
        return 1
        ;;
      *)
        health="$(aws eks describe-nodegroup --cluster-name "$EKS_CLUSTER" --nodegroup-name "$STATIC_EGRESS_NODEGROUP_NAME" \
          --region "$AWS_REGION" --query 'nodegroup.health.issues[0].message' --output text 2>/dev/null || echo "")"
        if [[ -n "$health" && "$health" != "None" ]]; then
          log "status=$st — $health"
        else
          log "status=$st ($(date +%H:%M:%S))"
        fi
        sleep 30
        ;;
    esac
  done
  log "Timed out after 30 minutes"
  return 1
}

if [[ -z "${STATIC_EGRESS_PRIVATE_SUBNET_ID:-}" ]]; then
  echo "ERROR: run ./provision-network.sh first (missing egress.env)." >&2
  exit 1
fi

if ! command -v aws >/dev/null 2>&1; then
  echo "ERROR: aws CLI not found." >&2
  exit 1
fi

ng_status() {
  aws eks describe-nodegroup --cluster-name "$EKS_CLUSTER" --nodegroup-name "$STATIC_EGRESS_NODEGROUP_NAME" \
    --region "$AWS_REGION" --query 'nodegroup.status' --output text 2>/dev/null || echo "MISSING"
}

print_failure() {
  aws eks describe-nodegroup --cluster-name "$EKS_CLUSTER" --nodegroup-name "$STATIC_EGRESS_NODEGROUP_NAME" \
    --region "$AWS_REGION" --query 'nodegroup.health.issues' --output table 2>/dev/null || true
}

create_nodegroup() {
  if ! aws iam get-role --role-name "$STATIC_EGRESS_NODE_ROLE_NAME" >/dev/null 2>&1; then
    echo "ERROR: IAM role $STATIC_EGRESS_NODE_ROLE_NAME missing." >&2
    echo "Run: ./deploy/aws/static-egress/ensure-node-prereqs.sh" >&2
    exit 1
  fi
  log "Creating node group $STATIC_EGRESS_NODEGROUP_NAME (subnet ${STATIC_EGRESS_PRIVATE_SUBNET_ID}, role ${STATIC_EGRESS_NODE_ROLE_NAME})..."
  aws eks create-nodegroup \
    --cluster-name "$EKS_CLUSTER" \
    --nodegroup-name "$STATIC_EGRESS_NODEGROUP_NAME" \
    --subnets "$STATIC_EGRESS_PRIVATE_SUBNET_ID" \
    --node-role "$NODE_ROLE_ARN" \
    --instance-types t3.small \
    --ami-type AL2023_x86_64_STANDARD \
    --scaling-config "minSize=1,maxSize=2,desiredSize=1" \
    --disk-size 20 \
    --labels "zorbahealth.io/egress=static" \
    --taints "key=zorbahealth.io/egress,value=static,effect=NO_SCHEDULE" \
    --tags "zorbahealth:component=static-egress" \
    --region "$AWS_REGION" \
    --output text --query 'nodegroup.nodegroupName'
  wait_nodegroup_active
}

status="$(ng_status)"
case "$status" in
  CREATE_FAILED|DELETE_FAILED)
    log "Node group $STATIC_EGRESS_NODEGROUP_NAME is $status:"
    print_failure
    echo "" >&2
    echo "Run: SKIP_CLUSTER_SUBNET_UPDATE=1 ./deploy/aws/static-egress/ensure-node-prereqs.sh" >&2
    echo "Then: ./deploy/aws/static-egress/create-nodegroup.sh" >&2
    exit 1
    ;;
  DELETING)
    log "Node group is DELETING; waiting until removed..."
    aws eks wait nodegroup-deleted --cluster-name "$EKS_CLUSTER" --nodegroup-name "$STATIC_EGRESS_NODEGROUP_NAME" \
      --region "$AWS_REGION" 2>/dev/null || true
    create_nodegroup
    ;;
  ACTIVE)
    log "Node group $STATIC_EGRESS_NODEGROUP_NAME is ACTIVE"
    ;;
  CREATING|UPDATING)
    log "Node group status=$status"
    wait_nodegroup_active
    ;;
  MISSING)
    create_nodegroup
    ;;
  *)
    log "Node group status=$status"
    wait_nodegroup_active
    ;;
esac

log "Nodes:"
log "  kubectl get nodes -l zorbahealth.io/egress=static"
log "Then: kubectl rollout restart deployment/notification-service -n dev"
log "VoIP.ms allowlist NAT_EIP=${NAT_EIP:-unknown}; verify checkip from the pod."
