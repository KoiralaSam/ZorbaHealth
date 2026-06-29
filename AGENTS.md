## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, invoke the `skill` tool with `skill: "graphify"` before doing anything else.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph content, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).

## EKS + Helm Deployment

This project uses Helm charts in `deploy/helm/` for **AWS EKS** deployment. The umbrella chart at `deploy/helm/Chart.yaml` deploys all services, infra (Postgres, Redis, RabbitMQ, Jaeger), and nginx-ingress via one command.

Full setup: [deploy/aws/README.md](deploy/aws/README.md).

### First-time EKS setup

```bash
cp deploy/aws/eks.env.example deploy/aws/eks.env
chmod +x deploy/aws/create-ecr-repos.sh deploy/aws/helm-install.sh
./deploy/aws/create-ecr-repos.sh

aws eks update-kubeconfig --region us-east-1 --name floral-bluegrass-sheepdog
kubectl apply -f deploy/kubernetes/development/secrets.yaml -n dev

helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm dependency build deploy/helm/

helm upgrade --install zorbahealth deploy/helm/ \
  --namespace dev \
  -f deploy/helm/values/eks-dev.yaml \
  --create-namespace
```

### Day-to-day dev (Tilt on EKS)

**Tilt owns namespace `dev`:** infra (Postgres, Redis, RabbitMQ, Jaeger), all services, web/mobile, **DB migrations** (`deploy/tilt/migrate-up.sh`), builds to **ECR**, port-forwards (5432, 8081, 3000, …). Manifests: `deploy/kubernetes/development/`.

One-time if Helm was installed on `dev`:

```bash
./deploy/tilt/switch-from-helm.sh
kubectl apply -f deploy/aws/eks-gp3-storageclass.yaml   # once per cluster (Auto Mode EBS)
```

Every session:

```bash
./deploy/tilt/preflight.sh
tilt up
```

Requires: `tilt`, `docker`, `kubectl`, `aws` CLI, `golang-migrate` (`brew install golang-migrate`). Optional override: `export DATABASE_URL=...` before `tilt up` (otherwise migrate reads `postgres-secret` from the cluster).

Do **not** run `helm install` on `dev` while using Tilt. CI still uses Helm on `develop`/`main`.

### CI/CD
GitHub Action at `.github/workflows/deploy.yml` builds changed services on push to `develop`/`main`, pushes to **ECR**, then runs `helm upgrade --install` on EKS. Set secret `AWS_DEPLOY_ROLE_ARN` (OIDC).

### Legacy Azure (AKS)
Scripts under `deploy/azure/` are retained for reference only.

### Services disabled in Helm
- `translation-model` (large LLM model, needs separate PVC provisioning)
- `translation-service` (depends on translation-model)
- Legacy: `health-provider-service`, `medical-records-service`, `rag-service` (no source code)
