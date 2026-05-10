# Q1 Journal Evaluation Plan

This document defines the implementation-facing evaluation goals needed to support a future Q1 journal submission for Zorba Health.

## Research framing

The target contribution statement is:

1. An open-source voice-based healthcare AI architecture
2. A controlled MCP-based tool gateway for healthcare agents
3. A FHIR-compatible RAG pipeline for patient-context retrieval
4. A safety, consent, and audit framework for healthcare AI interactions
5. A reproducible deployment and evaluation setup

## Evaluation principles

- prefer synthetic or de-identified data
- keep datasets reproducible
- separate engineering readiness from research claims
- measure safety, grounding, latency, and reproducibility explicitly

## Metrics to measure

The following metrics are part of the target evaluation scope and are intentionally named here now so later phases can implement them without ambiguity:

- average API latency
- average voice response latency
- RAG retrieval latency
- concurrent call handling
- RabbitMQ event processing delay
- retrieval precision
- retrieval recall
- answer groundedness
- hallucination rate
- unsafe medical response rate
- emergency escalation accuracy
- translation quality
- message retry success
- notification failure rate
- time to deploy locally
- documentation completeness

## Target datasets and scenario sources

- Synthea synthetic patient data
- sample FHIR records checked into the repository
- synthetic patient call scenarios
- emergency symptom scenarios
- translation test set
- RAG question-answer set

## Evidence expected from later phases

- architecture and service docs
- reproducible local deployment docs
- evaluation scripts under `scripts/evaluation`
- sample data under `examples/`
- automated tests aligned with safety, retrieval, and messaging behavior

## Ethics and data handling note

This project should use synthetic, demo, or otherwise non-production data for public evaluation artifacts unless a separate approved governance process exists.
