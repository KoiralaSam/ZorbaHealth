from __future__ import annotations

import logging
import os
from typing import Any

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

logger = logging.getLogger("zorba.observability")

_provider: TracerProvider | None = None


def configure_tracing(service_name: str = "voice-agent-service") -> TracerProvider:
    global _provider
    if _provider is not None:
        return _provider

    endpoint = os.environ.get("JAEGER_ENDPOINT", "http://localhost:4318/v1/traces").strip()
    environment = os.environ.get("ENVIRONMENT", "development").strip() or "development"

    resource = Resource.create(
        {
            "service.name": service_name,
            "deployment.environment": environment,
        }
    )
    provider = TracerProvider(resource=resource)
    provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter(endpoint=endpoint)))
    trace.set_tracer_provider(provider)

    _provider = provider
    logger.info("tracing initialized service=%s endpoint=%s", service_name, endpoint)
    return provider


def shutdown_tracing() -> None:
    if _provider is not None:
        _provider.shutdown()


def tracer(name: str) -> Any:
    return trace.get_tracer(name)
