#!/usr/bin/env bash
# Fix EKS Auto Mode when nodeRoleArn was set to the cluster role at enable time.
# Disables Auto Mode (if enabled), ensures AmazonEKSAutoNodeRole exists, re-enables Auto Mode.
#
# Run with admin credentials. Disabling terminates Auto Mode EC2 and deletes Auto Mode LBs.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck disable=SC1091
source "${EKS_ENV_FILE:-$ROOT/deploy/aws/eks.env}" 2>/dev/null || true

: "${AWS_REGION:=us-east-1}"
: "${EKS_CLUSTER:=floral-bluegrass-sheepdog}"
: "${NODE_ROLE_ARN:=arn:aws:iam::954976298234:role/AmazonEKSAutoNodeRole}"
export AWS_REGION AWS_DEFAULT_REGION="$AWS_REGION"
export PATH="${HOME}/.local/bin:${PATH}"

log() { printf '[fix-auto-mode] %s\n' "$*"; }

ensure_node_role_exists() {
  local role_name="${NODE_ROLE_ARN##*/}"
  if aws iam get-role --role-name "$role_name" >/dev/null 2>&1; then
    log "IAM role exists: $role_name"
    return
  fi
  log "Creating IAM role $role_name (required before re-enabling Auto Mode)..."
  aws iam create-role \
    --role-name "$role_name" \
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
    arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPullOnly; do
    aws iam attach-role-policy --role-name "$role_name" --policy-arn "$policy"
  done
  log "Created $NODE_ROLE_ARN"
}

log "Cluster: $EKS_CLUSTER  Target node role: $NODE_ROLE_ARN"
current="$(aws eks describe-cluster --name "$EKS_CLUSTER" --query 'cluster.computeConfig.nodeRoleArn' --output text 2>/dev/null || echo "null")"
log "Current nodeRoleArn: $current"

if [[ "$current" == "$NODE_ROLE_ARN" ]]; then
  log "Already correct. Run: aws eks update-kubeconfig --region $AWS_REGION --name $EKS_CLUSTER && kubectl get nodes"
  exit 0
fi

if [[ "$current" != "null" && "$current" != "None" && -n "$current" ]]; then
  log "Disabling EKS Auto Mode first..."
  aws eks update-cluster-config \
    --name "$EKS_CLUSTER" \
    --compute-config '{"enabled":false}' \
    --kubernetes-network-config '{"elasticLoadBalancing":{"enabled":false}}' \
    --storage-config '{"blockStorage":{"enabled":false}}'
  aws eks wait cluster-active --name "$EKS_CLUSTER"
else
  log "Auto Mode already disabled; skipping disable step."
fi

ensure_node_role_exists

log "Re-enabling Auto Mode with node IAM role..."
aws eks update-cluster-config \
  --name "$EKS_CLUSTER" \
  --compute-config "{\"enabled\":true,\"nodeRoleArn\":\"${NODE_ROLE_ARN}\",\"nodePools\":[\"general-purpose\",\"system\"]}" \
  --kubernetes-network-config '{"elasticLoadBalancing":{"enabled":true}}' \
  --storage-config '{"blockStorage":{"enabled":true}}'
aws eks wait cluster-active --name "$EKS_CLUSTER"

log "Done. Verify:"
log "  aws eks describe-cluster --name $EKS_CLUSTER --query cluster.computeConfig.nodeRoleArn"
log "  aws eks update-kubeconfig --region $AWS_REGION --name $EKS_CLUSTER"
log "  kubectl get nodes"
