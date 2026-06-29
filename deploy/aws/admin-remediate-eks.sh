#!/usr/bin/env bash
# Run with account-admin credentials (NOT the limited dev IAM user).
# Fixes: ECR repos, EKS Auto Mode node IAM role, optional dev-user policy.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "${EKS_ENV_FILE:-$ROOT/deploy/aws/eks.env}" 2>/dev/null || true

: "${AWS_REGION:=us-east-1}"
: "${AWS_ACCOUNT_ID:=954976298234}"
: "${EKS_CLUSTER:=floral-bluegrass-sheepdog}"
: "${NODE_ROLE_NAME:=AmazonEKSAutoNodeRole}"
: "${DEV_IAM_USER:=koiralas2}"
export AWS_REGION AWS_DEFAULT_REGION="$AWS_REGION"

NODE_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/${NODE_ROLE_NAME}"

# macOS official AWS CLI v2 installer (see deploy/aws/install-aws-cli-macos.sh)
export PATH="${HOME}/.local/bin:${PATH}"

log() { printf '[admin-remediate] %s\n' "$*"; }

require_admin() {
  if ! command -v aws >/dev/null 2>&1; then
    echo "ERROR: aws CLI not found. Install it, then ensure ~/.local/bin is on PATH:" >&2
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\"" >&2
    echo "  aws configure   # or: aws configure sso" >&2
    exit 1
  fi
  if ! aws sts get-caller-identity --region "$AWS_REGION" >/dev/null 2>&1; then
    echo "ERROR: AWS credentials not valid (sts get-caller-identity failed)." >&2
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\"" >&2
    echo "  export AWS_REGION=us-east-1 AWS_DEFAULT_REGION=us-east-1" >&2
    echo "  aws configure   # or refresh SSO: aws sso login --profile YOUR_PROFILE" >&2
    exit 1
  fi
  log "Caller: $(aws sts get-caller-identity --query Arn --output text --region "$AWS_REGION")"
  if ! aws ecr describe-repositories --max-results 1 --region "$AWS_REGION" >/dev/null 2>&1; then
    echo "ERROR: this identity lacks ECR/IAM admin rights (ecr:DescribeRepositories denied)." >&2
    echo "  Re-run with an administrator user/role (not the limited dev IAM user)." >&2
    exit 1
  fi
}

create_node_role() {
  if aws iam get-role --role-name "$NODE_ROLE_NAME" >/dev/null 2>&1; then
    log "IAM role exists: $NODE_ROLE_NAME"
    return
  fi
  log "Creating node IAM role: $NODE_ROLE_NAME"
  aws iam create-role \
    --role-name "$NODE_ROLE_NAME" \
    --assume-role-policy-document '{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": { "Service": "ec2.amazonaws.com" },
        "Action": "sts:AssumeRole"
      }]
    }'
  for policy in \
    arn:aws:iam::aws:policy/AmazonEKSWorkerNodeMinimalPolicy \
    arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPullOnly \
    arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore; do
    aws iam attach-role-policy --role-name "$NODE_ROLE_NAME" --policy-arn "$policy"
  done
  log "Node role ready: $NODE_ROLE_ARN"
}

fix_compute_node_role() {
  current="$(aws eks describe-cluster --name "$EKS_CLUSTER" \
    --query 'cluster.computeConfig.nodeRoleArn' --output text)"
  if [[ "$current" == "$NODE_ROLE_ARN" ]]; then
    log "EKS compute nodeRoleArn already correct"
    return
  fi
  log "Current nodeRoleArn: $current"
  log "Target nodeRoleArn: $NODE_ROLE_ARN"
  log "nodeRoleArn cannot be changed in-place on Auto Mode (AWS locks it after enable)."
  log "Recycling Auto Mode: disable, then re-enable with the correct node role..."

  aws eks update-cluster-config \
    --name "$EKS_CLUSTER" \
    --compute-config '{"enabled":false}' \
    --kubernetes-network-config '{"elasticLoadBalancing":{"enabled":false}}' \
    --storage-config '{"blockStorage":{"enabled":false}}'
  aws eks wait cluster-active --name "$EKS_CLUSTER"

  aws eks update-cluster-config \
    --name "$EKS_CLUSTER" \
    --compute-config "{\"enabled\":true,\"nodeRoleArn\":\"${NODE_ROLE_ARN}\",\"nodePools\":[\"general-purpose\",\"system\"]}" \
    --kubernetes-network-config '{"elasticLoadBalancing":{"enabled":true}}' \
    --storage-config '{"blockStorage":{"enabled":true}}'
  aws eks wait cluster-active --name "$EKS_CLUSTER"
  log "Auto Mode re-enabled; wait ~10m then: kubectl get nodes"
}

create_ecr() {
  log "Creating ECR repositories..."
  "$ROOT/deploy/aws/create-ecr-repos.sh"
}

attach_dev_user_policy() {
  local policy_name="ZorbaHealthEKSDeveloper"
  local policy_file="$ROOT/deploy/aws/iam-eks-developer-policy.json"
  local policy_arn="arn:aws:iam::${AWS_ACCOUNT_ID}:policy/${policy_name}"
  if [[ ! -f "$policy_file" ]]; then
    log "Skip dev user policy (missing $policy_file)"
    return
  fi
  if ! aws iam get-policy --policy-arn "$policy_arn" >/dev/null 2>&1; then
    log "Creating IAM policy $policy_name for user $DEV_IAM_USER"
    policy_arn="$(aws iam create-policy \
      --policy-name "$policy_name" \
      --policy-document "file://$policy_file" \
      --query 'Policy.Arn' --output text)"
  fi
  aws iam attach-user-policy --user-name "$DEV_IAM_USER" --policy-arn "$policy_arn" 2>/dev/null \
    && log "Attached $policy_name to $DEV_IAM_USER" \
    || log "Could not attach policy to $DEV_IAM_USER (may already be attached or user renamed)"
}

main() {
  require_admin
  create_node_role
  fix_compute_node_role
  create_ecr
  attach_dev_user_policy
  log "Done. Verify: kubectl get nodes && $ROOT/deploy/aws/helm-preflight.sh"
}

main "$@"
