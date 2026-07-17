# Observability

Zorba Health uses OpenTelemetry for traces and can be deployed with a lightweight LGTM-style local stack for metrics, dashboards, and debugging.

## Current components

- application tracing via `shared/tracing`
- Jaeger-compatible OTLP ingestion in development
- Grafana dashboard stubs under `deploy/observability/grafana/`

## Local stack

Observability support files live under `deploy/observability/`:

- `otel-collector-config.yaml`
- `prometheus/prometheus.yml`
- `grafana/bridged-interpretation-dashboard.json`

The Docker Compose path in `deploy/docker/docker-compose.yml` includes:

- OpenTelemetry Collector
- Prometheus
- Jaeger
- Grafana

## Trace story

The target end-to-end trace for a successful voice interaction is:

1. Call received
2. `voice-agent-service`
3. STT provider
4. LLM provider
5. MCP tool call
6. downstream service (`health-records-service`, `location-service`, etc.)
7. `audit-service`
8. TTS provider
9. notification or escalation side effects when applicable

Use these attributes consistently on spans where possible:

- `request_id`
- `correlation_id`
- `patient_id_hash` (never raw patient identifiers)
- `provider_name`
- `model_name`
- `tool_name`
- `channel`
- `session_id`

## Metrics to expose

The following metrics are part of the implementation contract and should appear in dashboards and Prometheus rules:

- request latency
- gRPC error rate
- RabbitMQ queue depth
- tool-call latency
- LLM response latency
- STT latency
- TTS latency
- RAG retrieval latency
- embedding generation time
- call duration
- emergency escalation count
- failed notification count

## Prometheus query examples

- API latency p95: `histogram_quantile(0.95, sum by (le) (rate(http_server_duration_bucket[5m])))`
- gRPC errors/sec: `sum(rate(grpc_server_handled_total{grpc_code!="OK"}[5m]))`
- queue depth: `rabbitmq_queue_messages`
- notification failures/sec: `sum(rate(traces_spanmetrics_calls_total{service_name="notification-service",status_code="STATUS_CODE_ERROR"}[5m]))`

## Contributor guidance

- Do not add raw PHI to logs, traces, metric labels, or Grafana dashboards.
- Prefer hashed patient identifiers and correlation IDs.
- Keep provider/model names in telemetry to support evaluation and debugging.
- When adding a new service, document its trace names and key metrics in the same pull request.
