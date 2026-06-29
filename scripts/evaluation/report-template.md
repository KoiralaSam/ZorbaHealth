# Evaluation Report Template

## Run Metadata

- Date:
- Commit:
- Environment:
- Seed dataset:

## Metrics

| Metric | Command | Output file | Notes |
| --- | --- | --- | --- |
| Average API latency | `node scripts/evaluation/demo-smoke.mjs patient-portal-smoke` | `demo-smoke.json` | gateway-facing checks |
| Average voice response latency | `k6 run scripts/evaluation/load/api-gateway.js` | `k6-api-gateway.json` | placeholder until full voice harness lands |
| RAG retrieval latency | `node scripts/evaluation/demo-smoke.mjs rag-groundedness-check` | `demo-smoke.json` | citations required |
| RabbitMQ event delay | TODO | TODO | add consumer timestamp diff output |
| Retrieval precision / recall | TODO | TODO | use `scripts/evaluation/datasets/rag-qrels.jsonl` |
| Groundedness / hallucination | TODO | TODO | compare answer to citations |
| Emergency escalation accuracy | `node scripts/evaluation/demo-smoke.mjs hospital-escalation-smoke` | `demo-smoke.json` | seeded escalations |
| Translation quality | TODO | TODO | compare to `translation-cases.jsonl` |
| Notification failure rate | TODO | TODO | derive from notification audit events |
| Local deploy time | manual | `compose-startup.txt` | capture `docker compose up` timing |
| Documentation completeness | manual rubric | `docs-rubric.md` | see README and docs set |

## Findings

### Strengths

-

### Regressions / Gaps

-

### Evidence mapping to research claims

1. Open-source voice architecture:
2. Controlled MCP gateway:
3. FHIR RAG:
4. Safety / consent / audit:
5. Reproducible deploy + evaluation:
