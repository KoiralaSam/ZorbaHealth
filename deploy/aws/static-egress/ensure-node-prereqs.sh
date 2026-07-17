#!/usr/bin/env bash
# IAM role + EKS access entry + (optional) cluster subnet for static-egress managed node group.
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
export AWS_REGION AWS_DEFAULT_REGION="$AWS_REGION"

NODE_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/${STATIC_EGRESS_NODE_ROLE_NAME}"
NODEGROUP_EC2_POLICY_ARN="arn:aws:eks::aws:cluster-access-policy/AmazonEKSNodegroupEC2Policy"
# EC2_LINUX entries get node-join permissions from EKS automatically; AssociateAccessPolicy is STANDARD-only.

log() { printf '[static-egress-prereqs] %s\n' "$*"; }
warn() { printf '[static-egress-prereqs] WARN: %s\n' "$*" >&2; }

require_admin() {
  if ! aws sts get-caller-identity --region "$AWS_REGION" >/dev/null 2>&1; then
    echo "ERROR: AWS credentials invalid." >&2
    exit 1
  fi
  log "Caller: $(aws sts get-caller-identity --query Arn --output text --region "$AWS_REGION")"
}

ensure_node_role() {
  if ! aws iam get-role --role-name "$STATIC_EGRESS_NODE_ROLE_NAME" >/dev/null 2>&1; then
    log "Creating IAM role $STATIC_EGRESS_NODE_ROLE_NAME"
    aws iam create-role \
      --role-name "$STATIC_EGRESS_NODE_ROLE_NAME" \
      --assume-role-policy-document '{
        "Version": "2012-10-17",
        "Statement": [{
          "Effect": "Allow",
          "Principal": { "Service": "ec2.amazonaws.com" },
          "Action": "sts:AssumeRole"
        }]
      }' >/dev/null
  fi
  for policy in \
    arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy \
    arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy \
    arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly; do
    aws iam attach-role-policy --role-name "$STATIC_EGRESS_NODE_ROLE_NAME" --policy-arn "$policy" 2>/dev/null \
      || true
  done
  log "Node role ready: $NODE_ROLE_ARN"
}

access_entry_exists() {
  aws eks describe-access-entry --cluster-name "$EKS_CLUSTER" --principal-arn "$NODE_ROLE_ARN" \
    --region "$AWS_REGION" >/dev/null 2>&1
}

ensure_access_entry() {
  if access_entry_exists; then
    log "Access entry exists for $STATIC_EGRESS_NODE_ROLE_NAME"
    return
  fi
  log "Creating EKS access entry (authenticationMode=API requires this)"
  if out="$(aws eks create-access-entry \
    --cluster-name "$EKS_CLUSTER" \
    --principal-arn "$NODE_ROLE_ARN" \
    --type EC2_LINUX \
    --region "$AWS_REGION" 2>&1)"; then
    echo "$out"
    return
  fi
  if [[ "$out" == *ResourceInUseException* ]] || [[ "$out" == *"already in use"* ]]; then
    log "Access entry already exists (describe may have been denied earlier)"
    return
  fi
  echo "$out" >&2
  exit 1
}

ensure_access_policy() {
  local entry_type
  entry_type="$(aws eks describe-access-entry --cluster-name "$EKS_CLUSTER" --principal-arn "$NODE_ROLE_ARN" \
    --region "$AWS_REGION" --query 'accessEntry.type' --output text 2>/dev/null || echo "UNKNOWN")"
  if [[ "$entry_type" == "EC2_LINUX" ]]; then
    log "Access entry type EC2_LINUX — EKS grants node join permissions automatically (no AssociateAccessPolicy)"
    return
  fi
  if [[ "$entry_type" == "STANDARD" ]]; then
    local associated
    associated="$(aws eks list-associated-access-policies \
      --cluster-name "$EKS_CLUSTER" \
      --principal-arn "$NODE_ROLE_ARN" \
      --region "$AWS_REGION" \
      --query "associatedAccessPolicies[?policyArn=='${NODEGROUP_EC2_POLICY_ARN}'].policyArn | [0]" \
      --output text 2>/dev/null || echo "None")"
    if [[ -n "$associated" && "$associated" != "None" ]]; then
      log "Access policy already associated: AmazonEKSNodegroupEC2Policy"
      return
    fi
    log "Associating AmazonEKSNodegroupEC2Policy on STANDARD access entry"
    aws eks associate-access-policy \
      --cluster-name "$EKS_CLUSTER" \
      --principal-arn "$NODE_ROLE_ARN" \
      --policy-arn "$NODEGROUP_EC2_POLICY_ARN" \
      --access-scope type=cluster \
      --region "$AWS_REGION"
    return
  fi
  warn "Could not read access entry type ($entry_type); skipping policy association"
}

ensure_cluster_subnet() {
  if [[ "${SKIP_CLUSTER_SUBNET_UPDATE:-0}" == "1" ]]; then
    log "Skipping cluster subnet update (SKIP_CLUSTER_SUBNET_UPDATE=1)"
    return 0
  fi
  local private_subnet="$STATIC_EGRESS_PRIVATE_SUBNET_ID"
  local vpc_json current_subnets public private
  vpc_json="$(aws eks describe-cluster --name "$EKS_CLUSTER" --region "$AWS_REGION" \
    --query 'cluster.resourcesVpcConfig' --output json)"
  current_subnets="$(echo "$vpc_json" | python3 -c 'import json,sys; print(",".join(json.load(sys.stdin)["subnetIds"]))')"
  if [[ ",$current_subnets," == *",$private_subnet,"* ]]; then
    log "Cluster already includes private subnet $private_subnet"
    return 0
  fi
  public="$(echo "$vpc_json" | python3 -c 'import json,sys; print(str(json.load(sys.stdin)["endpointPublicAccess"]).lower())')"
  private="$(echo "$vpc_json" | python3 -c 'import json,sys; print(str(json.load(sys.stdin)["endpointPrivateAccess"]).lower())')"
  log "Adding private subnet to cluster VPC config (needs eks:UpdateClusterConfig — often admin-only)"
  if ! aws eks update-cluster-config \
    --name "$EKS_CLUSTER" \
    --region "$AWS_REGION" \
    --resources-vpc-config "subnetIds=${current_subnets},${private_subnet},endpointPublicAccess=${public},endpointPrivateAccess=${private}" \
    --output text --query 'update.id'; then
    warn "Could not update cluster subnets. An admin can run:"
    warn "  SKIP_CLUSTER_SUBNET_UPDATE=0 AWS_PROFILE=<admin> $0"
    warn "Or set SKIP_CLUSTER_SUBNET_UPDATE=1 and continue (NAT egress may still work)."
    return 1
  fi
  log "Waiting for cluster update to complete..."
  aws eks wait cluster-active --name "$EKS_CLUSTER" --region "$AWS_REGION"
}

delete_failed_nodegroup() {
  local status
  status="$(aws eks describe-nodegroup --cluster-name "$EKS_CLUSTER" --nodegroup-name static-egress \
    --region "$AWS_REGION" --query 'nodegroup.status' --output text 2>/dev/null || echo "MISSING")"
  case "$status" in
    MISSING) log "No static-egress node group" ;;
    CREATE_FAILED|DELETING)
      log "Deleting node group static-egress (status=$status)..."
      aws eks delete-nodegroup --cluster-name "$EKS_CLUSTER" --nodegroup-name static-egress --region "$AWS_REGION"
      aws eks wait nodegroup-deleted --cluster-name "$EKS_CLUSTER" --nodegroup-name static-egress --region "$AWS_REGION"
      ;;
    ACTIVE)
      log "Node group static-egress is ACTIVE; skip delete"
      ;;
    *)
      log "Node group status=$status — delete manually if you need a clean recreate"
      ;;
  esac
}

main() {
  require_admin
  [[ -n "${STATIC_EGRESS_PRIVATE_SUBNET_ID:-}" ]] || { echo "ERROR: missing egress.env" >&2; exit 1; }
  ensure_node_role
  ensure_access_entry
  ensure_access_policy
  ensure_cluster_subnet || true
  if [[ "${DELETE_FAILED_NODEGROUP:-1}" == "1" ]]; then
    delete_failed_nodegroup
  fi
  log "Done. Re-run: ./deploy/aws/static-egress/create-nodegroup.sh"
}

main "$@"
