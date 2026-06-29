#!/usr/bin/env bash
# Admin-only: add static-egress private subnet to the EKS cluster API object.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${EKS_ENV_FILE:-$ROOT/deploy/aws/eks.env}" 2>/dev/null || true
# shellcheck disable=SC1091
source "$DIR/egress.env"

: "${AWS_REGION:=us-east-1}"
: "${EKS_CLUSTER:=floral-bluegrass-sheepdog}"
export AWS_REGION AWS_DEFAULT_REGION="$AWS_REGION"

SKIP_CLUSTER_SUBNET_UPDATE=0 DELETE_FAILED_NODEGROUP=0 "$DIR/ensure-node-prereqs.sh"
