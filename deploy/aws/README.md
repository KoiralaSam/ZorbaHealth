# AWS EKS deployment

Primary cluster: **`floral-bluegrass-sheepdog`** in **`us-east-1`** (account `954976298234`).

Helm charts live under `deploy/helm/`. Images are stored in **ECR** at `954976298234.dkr.ecr.us-east-1.amazonaws.com/<service>`.

**Region:** Scripts load `deploy/aws/eks.env` (copy from `eks.env.example`). If your AWS CLI default region is not `us-east-1`, keep `AWS_REGION=us-east-1` in that file so STS/ECR/EKS calls hit the right region.

## Migrate from AKS

1. Point kubeconfig at EKS: `aws eks update-kubeconfig --region us-east-1 --name floral-bluegrass-sheepdog`
2. Stop using `deploy/azure/` scripts and the old AKS context.
3. `./deploy/aws/create-ecr-repos.sh` (needs `ecr:CreateRepository`) and push `:latest` for each service (or use GitHub Actions on `develop`/`main`).
4. `kubectl apply -f deploy/kubernetes/development/secrets.yaml -n dev`
5. `./deploy/aws/helm-preflight.sh` then staged `./deploy/aws/helm-install.sh --infra-only --no-wait`, push images, full `./deploy/aws/helm-install.sh`
6. Local dev: ECR login + `tilt up` (same `dev` namespace — do not Helm-install infra while Tilt runs).

Legacy AKS material remains under `deploy/azure/` for reference only.

## One-time setup

0. **AWS CLI** — required before `aws eks update-kubeconfig`. On macOS, if `brew install awscli` gives `pyexpat` / `command not found: aws`, use the official v2 installer:

```bash
chmod +x deploy/aws/install-aws-cli-macos.sh
./deploy/aws/install-aws-cli-macos.sh
export PATH="$HOME/.local/bin:$PATH"
aws configure   # access key + secret, region us-east-1 — or use IAM Identity Center: aws configure sso
```

1. Install [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) and configure credentials (`aws configure` or SSO).
2. Copy env defaults and edit if your cluster name differs:

```bash
cp deploy/aws/eks.env.example deploy/aws/eks.env
```

3. Create ECR repositories (idempotent):

```bash
chmod +x deploy/aws/create-ecr-repos.sh deploy/aws/helm-install.sh
./deploy/aws/create-ecr-repos.sh
```

4. Merge Kubernetes secrets (never commit real `secrets.yaml`):

```bash
aws eks update-kubeconfig --region us-east-1 --name floral-bluegrass-sheepdog
kubectl create namespace dev --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f deploy/kubernetes/development/secrets.yaml -n dev
```

5. Install the stack:

```bash
./deploy/aws/helm-install.sh
```

Ingress uses the **ingress-nginx** subchart with a **LoadBalancer** Service. After install:

```bash
kubectl get svc -n dev -l app.kubernetes.io/name=ingress-nginx
```

Point DNS at the external hostname (or use `/etc/hosts` for `*.zorbahealth.dev` during testing).

### Storage

Postgres and RabbitMQ PVCs use **`storageClassName: gp2`** in `deploy/helm/values/eks-dev.yaml` for `floral-bluegrass-sheepdog` (only `gp2` is present today). After you install the [EBS CSI driver](https://docs.aws.amazon.com/eks/latest/userguide/ebs-csi.html) and a `gp3` StorageClass, switch `infra.postgres.storageClass` / `rabbitmq` back to `gp3`.

## Static egress IP (VoIP.ms SMS)

Auto Mode nodes use **public** subnets and **ephemeral** egress IPs. For a stable VoIP.ms REST allowlist, use **NAT + EIP + managed node group** and pin `notification-service`. Step-by-step: [deploy/aws/static-egress/README.md](static-egress/README.md).

## Local dev with Tilt (EKS)

Tilt builds images, pushes to ECR, and applies `deploy/kubernetes/development/` to the **`dev`** namespace on your current kube context.

```bash
aws eks update-kubeconfig --region us-east-1 --name floral-bluegrass-sheepdog
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin 954976298234.dkr.ecr.us-east-1.amazonaws.com
tilt up
```

Optional: override registry via `ECR_REGISTRY` before `tilt up`.

Do **not** run Helm infra and Tilt infra on the same namespace at once (duplicate Postgres/RabbitMQ).

## CI/CD (GitHub Actions)

Workflow: `.github/workflows/deploy.yml`

Required repository secrets:

| Secret | Purpose |
|--------|---------|
| `AWS_DEPLOY_ROLE_ARN` | IAM role for OIDC (`sts:AssumeRoleWithWebIdentity`) with ECR push + EKS deploy |

The role needs at least: `ecr:*` (or scoped push), `eks:DescribeCluster`, and `kubectl`-compatible access via `aws-auth` / EKS access entries for the GitHub OIDC principal.

Configure the [GitHub OIDC identity provider](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/configuring-openid-connect-in-amazon-web-services) in IAM, then trust it on the deploy role.

Helm values file for deploys: `deploy/helm/values/eks-dev.yaml`.

## Staged install (recommended first time)

Empty ECR repos are the #1 cause of `helm --wait` timeouts. Bring infra up first, push images, then the umbrella chart.

If **EC2 launches but `kubectl get nodes` stays empty** (nodeclaims stuck `Node not registered`), the cluster `computeConfig.nodeRoleArn` may point at the cluster service role instead of a dedicated **EC2 node role**. An account admin should run:

```bash
chmod +x deploy/aws/admin-remediate-eks.sh
# AWS_PROFILE=<admin> or admin access keys in the shell
./deploy/aws/admin-remediate-eks.sh
```

That script creates `zorbahealth-eks-auto-node`, updates EKS Auto Mode, creates ECR repos, and attaches `deploy/aws/iam-eks-developer-policy.json` to user `koiralas2` (edit the script if your dev user differs).

```bash
./deploy/aws/helm-preflight.sh
./deploy/aws/helm-install.sh --infra-only --no-wait
kubectl get pods,pvc -n dev
# push images to ECR, then:
./deploy/aws/helm-install.sh
```

Use `./deploy/aws/helm-install.sh --no-wait` on the full chart if you want Helm to return immediately and debug pods yourself.

## First-install errors (quick reference)

| Symptom | Likely cause | Fix |
|--------|----------------|-----|
| `helm` hangs then `timed out waiting for the condition` | App pods `ImagePullBackOff` (no `:latest` in ECR) | Push images or staged `--infra-only` first |
| PVC `Pending`, events mention `storageclass "gp3" not found` | No gp3 on cluster | `kubectl get sc`; use `gp2` or install [EBS CSI](https://docs.aws.amazon.com/eks/latest/userguide/ebs-csi.html) |
| `CreateContainerConfigError` / secret not found | Secrets not applied | `kubectl apply -f deploy/kubernetes/development/secrets.yaml -n dev` |
| `403` / `pull access denied` for ECR | Node IAM or wrong account | EKS Auto Mode usually OK; verify image URI account/region |
| `Insufficient cpu/memory` | Small node group | `kubectl describe nodes`; raise node count or trim `eks-dev.yaml` requests |
| `no nodes available to schedule pods` (EKS Auto Mode) | Node claims not joining | AWS Console → EKS → **Compute** / CloudWatch; check subnets, instance quotas, and node IAM role |
| Duplicate Postgres/RabbitMQ | Tilt + Helm same namespace | Use one path only on `dev` |

After a failed install:

```bash
kubectl get pods,pvc -n dev
kubectl get events -n dev --sort-by='.lastTimestamp' | tail -30
helm status zorbahealth -n dev
```

## Legacy Azure (AKS)

AKS scripts remain under `deploy/azure/` for reference only; they are not used by CI or Tilt defaults anymore.
