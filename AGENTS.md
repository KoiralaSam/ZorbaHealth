## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, invoke the `skill` tool with `skill: "graphify"` before doing anything else.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph content, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).

## Local development

**Primary path: Docker Compose.** See [docs/local-setup.md](docs/local-setup.md).

```bash
docker compose \
  -f deploy/docker/docker-compose.yml \
  -f deploy/docker/docker-compose.override.local.yml \
  up --build
```

**Optional: Tilt on local Kubernetes** (Docker Desktop / Minikube / kind) — not a cloud cluster.

```bash
cp deploy/kubernetes/development/secrets.example.yaml deploy/kubernetes/development/secrets.yaml
# fill placeholders, then:
./deploy/tilt/preflight.sh
tilt up
```

Optional packaging: Helm charts under `deploy/helm/` with [deploy/helm/values/dev.yaml](deploy/helm/values/dev.yaml). Do not run Helm and Tilt against the same `dev` namespace at once.

### Services disabled in Helm
- `translation-model` (large LLM model, needs separate PVC provisioning)
- `translation-service` (depends on translation-model)
- Legacy: `health-provider-service`, `medical-records-service`, `rag-service` (no source code)
