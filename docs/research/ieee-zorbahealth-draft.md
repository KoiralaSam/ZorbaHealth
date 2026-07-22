# Zorba Health: Consent-Aware Voice Healthcare AI with FHIR-Grounded RAG and an MCP Tool Gateway

> **Target venue:** IEEE Access (systems / biomedical informatics)  
> Corpus: `examples/evaluation-data/` · LaTeX: `docs/research/latex/main.tex`  
> Reproducibility seed: Synthea `1784242671144` · Draft freeze commit: `2d7429d0`

---

## Title

**Zorba Health: An Open-Source Voice-First Healthcare AI Architecture with Consent-Gated MCP Tooling and FHIR-Compatible Retrieval-Augmented Generation**

## Authors

Samarpan Koirala<sup>1</sup>  
<sup>1</sup>Independent Researcher; Maintainer, Zorba Health Open Source Project  
Email: samarpankoirala@gmail.com  

## Abstract

Voice interfaces can broaden access to digital health support, yet conversational agents that touch clinical context introduce risks of privacy leakage, missing consent, hallucinated advice, and uncontrolled tool use. We present **Zorba Health**, an open-source microservice platform that combines (i) a LiveKit-backed voice agent, (ii) a Model Context Protocol (MCP) tool gateway that mediates agent access to backends, (iii) a FHIR-compatible ingestion and retrieval-augmented generation (RAG) path that returns citations, and (iv) an append-only audit and consent service that gates sensitive operations. We release a synthetic evaluation corpus of **16 patients** (**646** filtered FHIR resources), **25** gold QA items, **22** exact plus **22** paraphrase escalation utterances, and **8** consent scenarios. Offline type-aware lexical **Recall@3 = 1.00**. Exact-phrase emergency escalation **F1 = 1.00**, but paraphrase **F1 = 0.29** (combined **F1 = 0.71**), exposing brittleness of rule-based safety. Consent-gate e2e on record QA achieves **deny/allow/regrant = pass** (known gap: third-party model consent not yet enforced in RAG). A live OpenAI RAG evaluation with **answer-string-required** scoring yields **answerable accuracy = 0.85** (17/20), **citation presence = 1.00**, **unanswerable safe refusal = 1.00** (5/5), mean latency **1.67 s** (p95 **2.43 s**). We argue that separating voice runtime, tool policy, clinical retrieval, and compliance logging is a practical path toward auditable healthcare agents.

**Index Terms**—Healthcare AI, voice agents, FHIR, retrieval-augmented generation, Model Context Protocol, consent, audit, microservices, Synthea, IEEE Access.

---

## I. Introduction

### A. Motivation

Telephone and voice channels remain highly accessible for patients who struggle with app-centric portals. Large language models (LLMs) enable natural dialogue, but naïve deployments can invent clinical facts, bypass consent, omit audit trails, and couple the agent directly to sensitive stores and third-party APIs. The Model Context Protocol (MCP) standardizes tool exposure but explicitly cannot enforce consent or authorization at the protocol layer; implementors must build those controls themselves [MCP Spec, 2025]. WHO guidance on large multi-modal models for health similarly emphasizes governance, accountability, and careful escalation [WHO, 2024].

### B. Problem statement

How can a voice healthcare assistant be engineered so that tool use, record retrieval, and AI summarization are **consent-gated**, **auditable**, **grounded in patient-specific FHIR context**, and **reproducibly evaluable**?

### C. Contributions

1. **Open-source voice healthcare architecture** — Go microservices, Next.js surfaces, gRPC, RabbitMQ, PostgreSQL (pgvector), Redis, LiveKit voice integration.
2. **Controlled MCP tool gateway** — agents invoke healthcare tools only through `mcp-server`, materializing the consent flows the MCP specification leaves to implementors.
3. **FHIR-compatible RAG pipeline** — ingest R4 bundles, chunk/embed, retrieve with filters/rerank, summarize over retrieved chunks only, return citations.
4. **Safety, consent, and audit framework** — `audit-service` with grant/revoke history and append-only events; voice-side emergency escalation.
5. **Reproducible synthetic evaluation package** — corpus, offline baselines, live RAG harness, consent-gate tests, and paraphrase safety stress test.

### D. Organization

Section II reviews related work. Section III describes the system and standards alignment. Section IV details data collection. Section V presents experiments. Section VI discusses limitations and threats. Section VII concludes.

---

## II. Related Work

**Voice and conversational health systems.** Relational agents and virtual nurses have long targeted low-literacy and telephone-accessible care [Bickmore et al., 2009]. Modern LLM medical systems encode substantial clinical knowledge [Singhal et al., 2023; Nori et al., 2023], but many demos treat the dialogue model as the system of record. Polaris [Mukherjee et al., 2024] is the closest published voice healthcare agent: a proprietary multi-agent constellation evaluated by over 1,100 U.S. licensed nurses and 130 physicians, reporting aggregate parity with human nurses on conversational and safety axes. Zorba Health differs on three axes: (i) it is fully open-source with reproducible synthetic evaluation; (ii) it centers consent-gated MCP mediation and append-only audit rather than constellation training; and (iii) it makes no clinical-parity claims.

**FHIR and synthetic EHR.** HL7 FHIR R4 is the dominant exchange model for interoperable health apps [Mandl et al., 2015; HL7 FHIR R4]. Synthea generates longitudinal synthetic patients widely used for methods research without PHI [Walonoski et al., 2018]. MedAgentBench [Jiang et al., 2025] further provides a FHIR-compliant interactive EHR environment (100 patients, 300 clinician-authored tasks) for benchmarking medical LLM agents—an evaluation target complementary to our synthetic corpus.

**RAG for clinical QA.** Retrieval-augmented generation grounds LLMs in external evidence [Lewis et al., 2020]. Medical RAG benchmarks and automatic faithfulness metrics (e.g., RAGAS) quantify hallucination risk [Es et al., 2023; Xiong et al., 2024]. Almanac [Zakka et al., 2024] shows clinician-panel preference for guideline-grounded RAG over ungrounded chatbots on factuality and adversarial safety. FHIR-RAG-MEDS [Kabak et al., 2026] integrates HL7 FHIR with RAG for personalized decision support over clinical guidelines. Our pipeline likewise enforces patient-scoped vector search and consent checks before summarization, and returns citations with answers, but targets patient-record QA rather than guideline retrieval.

**Agent tool use and MCP.** Tool-using LLMs (ReAct, Toolformer) show that models can call external actions [Yao et al., 2023; Schick et al., 2023]. AgentClinic [Schmidgall et al., 2024] and HealthBench [Arora et al., 2025] stress that interactive, open-ended clinical evaluation is harder than static QA; HealthBench further includes physician rubrics spanning emergency referral and safety behaviors. The Model Context Protocol (MCP) standardizes tool exposure to agents [MCP Spec, 2025]. Early healthcare MCP prototypes document privacy and compliance layers [Shehab, 2025; ElSayed et al., 2025], while Hou et al. [2025] systematize MCP threat scenarios (tool poisoning, unauthorized access, installer spoofing). The MCP specification itself states that the protocol cannot enforce consent or authorization at the protocol layer and that implementors SHOULD build robust consent flows [MCP Spec, 2025]. We treat MCP as a mandatory mediation layer and materialize consent and audit as first-class services around it.

**Consent, audit, and crisis escalation.** HIPAA [1996] and WHO guidance on large multi-modal models for health [WHO, 2024] motivate purpose limitation, access control, accountable logging, and careful escalation paths. Break-glass emergency patterns further require explicit detection of crisis language. PsyCrisisBench [Deng et al., 2025] shows LLMs reaching F1 ≈ 0.88 on suicidal-ideation detection in psychological-hotline transcripts, while Arnaiz-Rodriguez et al. [2026] find that all evaluated models still struggle with indirect crisis signals. These results frame our paraphrase-F1 collapse (0.29) as a known failure mode of surface-form safety rules and motivate LLM- or embedding-based escalation as future work. Zorba Health materializes consent and audit as a dedicated service with typed consents checked before record retrieval, summarization, and location tools.

---

## III. System Architecture

### A. Overview

| Layer | Components |
| --- | --- |
| Clients | Next.js patient/hospital UI, mobile MVP, phone via FreePBX / LiveKit SIP |
| Edge | `api-gateway`, `voice-agent-service` |
| Tool mediation | `mcp-server` |
| Domain | patient, auth, health-records, location, notification, translation, analytics |
| Compliance | `audit-service` |
| Data plane | PostgreSQL + pgvector, Redis, RabbitMQ |
| Observability | OpenTelemetry, Jaeger, Prometheus, Grafana |

Voice sessions do **not** query FHIR stores directly. The agent requests tools via MCP; MCP calls gRPC services and records audit outcomes. See Fig. 1–2 in the LaTeX camera-ready.

### B. Consent-aware RAG

Implemented in `services/health-records-service/internal/rag`:

1. Validate patient ID and query.  
2. Check `HEALTH_RECORD_ACCESS`; on denial, audit and abort.  
3. Embed query (default `text-embedding-3-small`).  
4. Patient-scoped vector search with over-fetch, metadata filter, light rerank.  
5. Optional LLM summarization **only** over retrieved chunk texts.  
6. Audit `HEALTH_RECORD_SEARCHED` / `HEALTH_RECORD_SUMMARIZED`.  
7. Return answer + citations.

### C. Safety escalation

`voice-agent-service` applies a deterministic phrase classifier (`safety.py`) for emergency-like utterances, triggers hospital-visible escalation, and emits `EMERGENCY_ESCALATION_TRIGGERED` audit events.

### D. Standards alignment

Zorba Health uses application-level consent types (`HEALTH_RECORD_ACCESS`, `THIRD_PARTY_MODEL_PROCESSING`, and related grants) checked by `audit-service` before PHI-touching tools run. These types are complementary to, but not yet mapped onto, the FHIR R4 Consent resource [HL7 FHIR Consent], which encodes provision, purpose, and actor in a portable EHR-native form; mapping typed grants to FHIR Consent is future interoperability work. Similarly, SMART App Launch v2.2 scopes (e.g., `patient/*.rs`) [SMART App Launch] remain the incumbent pattern for authorizing FHIR API clients; our MCP gateway sits *above* SMART-style resource scopes as a tool-mediation and audit boundary for agent invocations, not as a replacement for OAuth/SMART authorization. WHO guidance on large multi-modal models for health [WHO, 2024] informs our ethics framing: synthetic-only evaluation, no clinical-efficacy claims, and explicit escalation paths.

### E. Reproducible deployment

Docker Compose (`deploy/docker/docker-compose.yml`) and Kubernetes/Tilt; migrations under `migrations/`.

---

## IV. Data Collection and Datasets

### A. Principles

Prefer synthetic data; keep regeneration deterministic; separate engineering readiness from clinical claims.

### B. Corpus (Table I resources)

| Split | Count |
| --- | ---: |
| Curated gold patients | 5 |
| Synthea-slim patients | 10 |
| Demo bundle | 1 |
| **Patients total** | **16** |
| FHIR resources (filtered) | **646** |
| Gold QA | **25** |
| Escalation exact / paraphrase | **22 / 22** |
| Consent scenarios | **8** |

Synthea seed `1784242671144` (Massachusetts, n=10). Slim bundles only are checked in (`examples/evaluation-data/`).

### C. Ethics

All public evaluation artifacts are **synthetic**. No production PHI. Future real-patient studies would require separate IRB/governance. This work does not constitute medical advice or a clinical trial.

---

## V. Experimental Setup and Results

### A. Research questions

- **RQ1:** Lexical / live RAG fact recovery on curated gold QA  
- **RQ2:** Exact vs paraphrase emergency escalation  
- **RQ3:** Consent grant/revoke gate for record QA  
- **RQ4:** Live embedding+LLM latency  

### B. Methods

1. Existence oracle + type-aware lexical Recall@3  
2. Live OpenAI RAG harness (`cmd/eval-live-rag`) with **answer-string-required** audit (`audit_live_rag_results.py`)  
3. Consent stub e2e mirroring `rag.Pipeline` deny/allow (`score_consent_e2e.py`)  
4. Exact + paraphrase safety scoring (`score_safety_paraphrase.py`)  
5. Go unit tests for FHIR validate + RAG mocks  

### C. Results

#### Table II — Lexical retrieval proxy

| Metric | Value |
| --- | ---: |
| Existence oracle recall | **1.00** |
| Type-aware Recall@3 | **1.00** |

#### Table III — Live OpenAI RAG (answer-string-required audit)

| Metric | Value |
| --- | ---: |
| Answerable accuracy | **0.85** (17/20) |
| Citation presence | **1.00** |
| Unanswerable safe refusal | **1.00** (5/5) |
| Mean / p95 latency | **1667 / 2432 ms** |

**Error analysis.** Failures concentrate on allergy questions (3 missed allergy facts in the answer text despite citations) and one medication answer that incorrectly listed penicillin (an allergy) alongside albuterol. These motivate stronger answer-side constraints (resource-type-aware prompts / citation-forced decoding) as future work. Earlier citation-only scoring overstated accuracy at 1.00; we report the tightened metric.

#### Table IV — Emergency escalation

| Split | n | Precision | Recall | F1 |
| --- | ---: | ---: | ---: | ---: |
| Exact phrases | 22 | 1.00 | 1.00 | **1.00** |
| Paraphrases | 22 | 1.00 | 0.17 | **0.29** |
| Combined | 44 | 1.00 | 0.55 | **0.71** |

Exact-set perfection is a regression sanity check. Paraphrase recall collapse shows the current classifier is **not** clinically robust under rewording.

#### Table V — Consent gate e2e

| Check | Result |
| --- | --- |
| Record QA deny when `HEALTH_RECORD_ACCESS` revoked | Pass |
| Record QA allow when granted | Pass |
| Revoke then re-grant | Pass |
| Scenario suite pass rate | **11/11** (1 known gap documented) |
| Known gap | `THIRD_PARTY_MODEL_PROCESSING` not enforced inside RAG pipeline |

#### Table VI — Engineering validation

| Check | Result |
| --- | --- |
| FHIR validation tests | Pass |
| RAG citation path (mocked) | Pass |
| Live RAG harness | Pass (Table III) |

---

## VI. Discussion

### A. Design implications

MCP + audit as first-class services create a single policy enforcement point shared by voice and web. Relative to Hou et al.'s MCP threat taxonomy [Hou et al., 2025], the gateway and audit path mitigate unauthorized tool invocation (deny-by-default consent gates), weak accountability (append-only event log), and unmediated PHI access (no direct store queries from the voice runtime). They do **not** yet address installer spoofing, malicious third-party MCP server listings, or cryptographic manifest signing for tool descriptors—orthogonal hardening left to deployment and future work. Relative to Polaris [Mukherjee et al., 2024], we trade proprietary constellation training and clinician-parity claims for an auditable open architecture and reproducible synthetic metrics.

### B. Standards and interoperability

Custom consent types currently optimize for pipeline clarity over EHR portability. Aligning with FHIR Consent [HL7 FHIR Consent] and composing MCP mediation with SMART App Launch scopes [SMART App Launch] would ease adoption in SMART-on-FHIR environments without abandoning the agent tool boundary.

### C. Limitations

- Curated live RAG set is small (5 patients / 25 QA).  
- Phrase safety fails under paraphrase; PsyCrisisBench and related crisis-handling studies [Deng et al., 2025; Arnaiz-Rodriguez et al., 2026] suggest LLM/embedding classifiers as the credible next step.  
- Third-party model consent is modeled but not yet enforced in RAG.  
- No clinician-rated groundedness (Almanac-/HealthBench-style rubrics); no real PHI.  
- Voice/SIP often external to Compose.  
- Not yet evaluated on MedAgentBench-style interactive FHIR agent tasks [Jiang et al., 2025].

### D. Threats to validity

Synthetic FHIR under-represents messy notes. Local latency underestimates telephony RTT. Stub consent e2e validates pipeline contract, not full HTTP portal auth.

---

## VII. Conclusion

Zorba Health provides an open, consent-aware architecture for voice healthcare agents with FHIR-grounded RAG and MCP-mediated tools. On synthetic evaluation, lexical retrieval is intact, live RAG answer-string accuracy is **0.85** with perfect citation presence and safe refusal on unanswerable LDL probes, consent deny/allow works for health-record access, and paraphrase stress testing honestly shows safety-classifier brittleness (**F1 0.29**). Future work: paraphrase-robust (LLM/embedding-based) escalation, third-party consent enforcement inside RAG, FHIR Consent / SMART scope alignment, MedAgentBench and HealthBench-style evaluation, Synthea-scale gold labels, and clinician review.

---

## Data and Code Availability

- Code: this open-source repository.  
- Evaluation data: `examples/evaluation-data/` (`DATASET_CARD.md`).  
- Reproduce:  
  `python3 scripts/evaluation/build_eval_corpus.py`  
  `python3 scripts/evaluation/score_keyword_retrieval.py`  
  `python3 scripts/evaluation/audit_live_rag_results.py`  
  `python3 scripts/evaluation/score_consent_e2e.py`  
  `python3 scripts/evaluation/score_safety_paraphrase.py`  
  `go run ./services/health-records-service/cmd/eval-live-rag ...` (requires `OPENAI_API_KEY`)  
- Synthea seed: `1784242671144`. Draft freeze commit: `2d7429d0`.

## Acknowledgments

No external funding was received for this draft. The author thanks open-source communities behind Synthea, FHIR, LiveKit, and MCP.

## Conflict of Interest

The author is the maintainer of the Zorba Health open-source project. No competing financial interests are declared.

## Ethics Statement

Evaluation uses synthetic data only (handcrafted + Synthea). No human subjects or production PHI were used. Outputs must not be interpreted as clinical advice. Framing follows WHO guidance on large multi-modal models for health [WHO, 2024]: governance, accountability, and no undeclared clinical claims.

## References

See `docs/research/latex/refs.bib` (IEEEtran).

---

## Appendix A — Submission checklist

- [x] Synthetic corpus + gold labels  
- [x] Offline + live RAG (audited) metrics  
- [x] Consent e2e + paraphrase safety  
- [x] IEEE Access framing + COI/funding/ethics  
- [x] Cover letter synced with live numbers  
- [x] Full LaTeX + figures + PDF  
- [x] Reproducibility freeze notes  
- [ ] Optional: institutional affiliation upgrade  
- [ ] Optional: clinician spot-check (future work)

## Appendix B — Cover letter blurb

We submit a systems paper on Zorba Health, an open-source voice healthcare platform that mediates LLM tool use through a consent- and audit-aware MCP gateway and answers patient questions with FHIR-compatible RAG and citations. We release a synthetic evaluation corpus (16 patients, 646 resources) and report: lexical Recall@3 = 1.00; live OpenAI RAG answer-string accuracy = 0.85 with citation presence = 1.00 and safe refusal on unanswerable LDL probes (mean 1.67 s); consent deny/allow/regrant pass for health-record access; and an honest paraphrase stress test where emergency-escalation F1 falls from 1.00 (exact) to 0.29 (paraphrase). The contribution is an auditable architecture and reproducible evaluation path, not a claim of clinical efficacy.
