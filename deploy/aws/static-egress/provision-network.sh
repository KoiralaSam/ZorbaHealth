#!/usr/bin/env bash
# Creates NAT Gateway + EIP + private subnet for static egress in the EKS cluster VPC.
# Run with account-admin credentials.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
# shellcheck disable=SC1091
source "${EKS_ENV_FILE:-$ROOT/deploy/aws/eks.env}" 2>/dev/null || true

: "${AWS_REGION:=us-east-1}"
: "${EKS_CLUSTER:=floral-bluegrass-sheepdog}"
# Public cluster subnet to host the NAT Gateway (must be MapPublicIpOnLaunch=true).
: "${STATIC_EGRESS_NAT_AZ:=us-east-1a}"
# Optional: set STATIC_EGRESS_PRIVATE_CIDR (e.g. 10.0.250.0/24) if auto-pick collides.
export AWS_REGION AWS_DEFAULT_REGION="$AWS_REGION"

OUT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_ENV="$OUT_DIR/egress.env"

log() { printf '[static-egress] %s\n' "$*"; }

require_admin() {
  if ! command -v aws >/dev/null 2>&1; then
    echo "ERROR: aws CLI not found." >&2
    exit 1
  fi
  if ! aws sts get-caller-identity --region "$AWS_REGION" >/dev/null 2>&1; then
    echo "ERROR: AWS credentials invalid." >&2
    exit 1
  fi
  log "Caller: $(aws sts get-caller-identity --query Arn --output text --region "$AWS_REGION")"
}

cluster_vpc() {
  aws eks describe-cluster --name "$EKS_CLUSTER" --region "$AWS_REGION" \
    --query 'cluster.resourcesVpcConfig.vpcId' --output text
}

pick_public_subnet() {
  local vpc="$1"
  local az="$2"
  aws ec2 describe-subnets --region "$AWS_REGION" \
    --filters "Name=vpc-id,Values=$vpc" "Name=availability-zone,Values=$az" \
    --query 'Subnets[?MapPublicIpOnLaunch==`true`].SubnetId | [0]' --output text
}

vpc_primary_cidr() {
  aws ec2 describe-vpcs --region "$AWS_REGION" --vpc-ids "$1" \
    --query 'Vpcs[0].CidrBlock' --output text
}

cidr_in_use() {
  local cidr="$1"
  local vpc="$2"
  local found
  found="$(aws ec2 describe-subnets --region "$AWS_REGION" \
    --filters "Name=vpc-id,Values=$vpc" \
    --query "Subnets[?CidrBlock==\`$cidr\`].SubnetId | [0]" --output text)"
  [[ -n "$found" && "$found" != "None" ]]
}

suggest_private_cidr() {
  local vpc_cidr="$1"
  local vpc="$2"
  local base prefix candidate
  base="${vpc_cidr%%/*}"
  IFS=. read -r o1 o2 o3 _ <<<"$base"
  for suffix in 250 251 252 248 240; do
    candidate="${o1}.${o2}.${suffix}.0/24"
    if ! cidr_in_use "$candidate" "$vpc"; then
      echo "$candidate"
      return 0
    fi
  done
  echo "ERROR: could not find free /24 in VPC; set STATIC_EGRESS_PRIVATE_CIDR" >&2
  exit 1
}

tag_subnet_for_eks() {
  local subnet="$1"
  aws ec2 create-tags --region "$AWS_REGION" --resources "$subnet" --tags \
    "Key=Name,Value=zorbahealth-static-egress-private" \
    "Key=kubernetes.io/cluster/${EKS_CLUSTER},Value=shared" \
    "Key=kubernetes.io/role/internal-elb,Value=1" \
    "Key=zorbahealth.io/egress,Value=static"
}

main() {
  require_admin

  local vpc nat_public_subnet private_cidr eip_alloc nat_id private_subnet rt_id
  vpc="$(cluster_vpc)"
  log "Cluster VPC: $vpc"

  nat_public_subnet="$(pick_public_subnet "$vpc" "$STATIC_EGRESS_NAT_AZ")"
  if [[ -z "$nat_public_subnet" || "$nat_public_subnet" == "None" ]]; then
    echo "ERROR: no public subnet in AZ $STATIC_EGRESS_NAT_AZ for VPC $vpc" >&2
    exit 1
  fi
  log "NAT public subnet ($STATIC_EGRESS_NAT_AZ): $nat_public_subnet"

  if [[ -n "${STATIC_EGRESS_PRIVATE_CIDR:-}" ]]; then
    private_cidr="$STATIC_EGRESS_PRIVATE_CIDR"
  else
    private_cidr="$(suggest_private_cidr "$(vpc_primary_cidr "$vpc")" "$vpc")"
  fi
  if cidr_in_use "$private_cidr" "$vpc"; then
    echo "ERROR: subnet CIDR $private_cidr already in use" >&2
    exit 1
  fi
  log "Private subnet CIDR: $private_cidr"

  # Reuse NAT if we already created one (tag Name=zorbahealth-static-egress-nat).
  nat_id="$(aws ec2 describe-nat-gateways --region "$AWS_REGION" \
    --filter "Name=vpc-id,Values=$vpc" "Name=tag:Name,Values=zorbahealth-static-egress-nat" \
      "Name=state,Values=available,pending" \
    --query 'NatGateways[0].NatGatewayId' --output text 2>/dev/null || true)"
  if [[ -z "$nat_id" || "$nat_id" == "None" ]]; then
    log "Allocating Elastic IP..."
    eip_alloc="$(aws ec2 allocate-address --region "$AWS_REGION" --domain vpc --query AllocationId --output text)"
    log "Creating NAT Gateway (this takes a few minutes)..."
    nat_id="$(aws ec2 create-nat-gateway --region "$AWS_REGION" \
      --subnet-id "$nat_public_subnet" --allocation-id "$eip_alloc" \
      --tag-specifications "ResourceType=natgateway,Tags=[{Key=Name,Value=zorbahealth-static-egress-nat}]" \
      --query NatGateway.NatGatewayId --output text)"
    aws ec2 wait nat-gateway-available --region "$AWS_REGION" --nat-gateway-ids "$nat_id"
  else
    log "Reusing NAT Gateway: $nat_id"
  fi

  local nat_eip
  nat_eip="$(aws ec2 describe-nat-gateways --region "$AWS_REGION" --nat-gateway-ids "$nat_id" \
    --query 'NatGateways[0].NatGatewayAddresses[0].PublicIp' --output text)"
  log "NAT public IP (VoIP.ms allowlist): $nat_eip"

  private_subnet="$(aws ec2 describe-subnets --region "$AWS_REGION" \
    --filters "Name=vpc-id,Values=$vpc" "Name=tag:Name,Values=zorbahealth-static-egress-private" \
    --query 'Subnets[0].SubnetId' --output text)"
  if [[ -z "$private_subnet" || "$private_subnet" == "None" ]]; then
    log "Creating private subnet..."
    private_subnet="$(aws ec2 create-subnet --region "$AWS_REGION" \
      --vpc-id "$vpc" --availability-zone "$STATIC_EGRESS_NAT_AZ" \
      --cidr-block "$private_cidr" \
      --tag-specifications "ResourceType=subnet,Tags=[{Key=Name,Value=zorbahealth-static-egress-private}]" \
      --query Subnet.SubnetId --output text)"
    aws ec2 modify-subnet-attribute --region "$AWS_REGION" \
      --subnet-id "$private_subnet" --no-map-public-ip-on-launch
    tag_subnet_for_eks "$private_subnet"
  else
    log "Reusing private subnet: $private_subnet"
    tag_subnet_for_eks "$private_subnet"
  fi

  rt_id="$(aws ec2 describe-route-tables --region "$AWS_REGION" \
    --filters "Name=association.subnet-id,Values=$private_subnet" \
    --query 'RouteTables[0].RouteTableId' --output text)"
  if [[ -z "$rt_id" || "$rt_id" == "None" ]]; then
    log "Creating route table (0.0.0.0/0 -> NAT)..."
    rt_id="$(aws ec2 create-route-table --region "$AWS_REGION" --vpc-id "$vpc" \
      --tag-specifications "ResourceType=route-table,Tags=[{Key=Name,Value=zorbahealth-static-egress-private-rt}]" \
      --query RouteTable.RouteTableId --output text)"
    aws ec2 create-route --region "$AWS_REGION" --route-table-id "$rt_id" \
      --destination-cidr-block 0.0.0.0/0 --nat-gateway-id "$nat_id" >/dev/null
    aws ec2 associate-route-table --region "$AWS_REGION" \
      --route-table-id "$rt_id" --subnet-id "$private_subnet" >/dev/null
  else
    log "Route table already associated: $rt_id"
  fi

  cat >"$OUT_ENV" <<EOF
# Generated by provision-network.sh — do not commit (gitignored).
AWS_REGION=$AWS_REGION
EKS_CLUSTER=$EKS_CLUSTER
STATIC_EGRESS_VPC_ID=$vpc
STATIC_EGRESS_NAT_AZ=$STATIC_EGRESS_NAT_AZ
STATIC_EGRESS_NAT_SUBNET_ID=$nat_public_subnet
STATIC_EGRESS_NAT_GATEWAY_ID=$nat_id
STATIC_EGRESS_PRIVATE_SUBNET_ID=$private_subnet
STATIC_EGRESS_PRIVATE_CIDR=$private_cidr
NAT_EIP=$nat_eip
EOF
  log "Wrote $OUT_ENV"
  log "Next: ./deploy/aws/static-egress/create-nodegroup.sh"
}

main "$@"
