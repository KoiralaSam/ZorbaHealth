# Zorba Health: Consent-Aware Voice Healthcare AI with FHIR-Grounded RAG and an MCP Tool Gateway

> **Venue:** GAISS conference (IEEEtran `conference` class, max six pages including references)  
> Corpus: `examples/evaluation-data/` · LaTeX: `docs/research/latex/main.tex`  
> Reproducibility seed: Synthea `1784242671144` · Draft freeze commit: `2d7429d0`

---

## Title

**Zorba Health: An Open-Source Voice-First Healthcare AI Architecture with Consent-Gated MCP Tooling and FHIR-Compatible Retrieval-Augmented Generation**

## Authors

Samarpan Koirala  
Email: samarpankoirala@gmail.com  

## Abstract

Voice interfaces can make digital health support more accessible, but healthcare agents can expose private data, miss consent requirements, generate unsupported advice, or use tools without adequate controls. Prior systems such as Polaris are proprietary and cannot be independently audited, and the Model Context Protocol (MCP) standardizes tool access but does not enforce consent or authorization. We present **Zorba Health**, an open-source microservice platform for voice healthcare agents. It combines a LiveKit voice agent, an MCP gateway for controlled tool access, a FHIR-compatible retrieval-augmented generation (RAG) system with citations, and an append-only audit and consent service. We evaluate the platform on a released synthetic corpus of **16 patients**, **646** filtered FHIR resources, **25** question-answer items, **44** emergency-escalation utterances, and **8** consent scenarios. In offline retrieval tests, the correct record appeared in the top three results for every query (**Recall@3 = 1.00**). The phrase-based escalation rule scored a perfect F1 of **1.00** on exact trigger phrases but only **0.29** on paraphrases (combined **0.71**), showing that rule-based safety is brittle. End-to-end (e2e) consent tests confirmed that record access is denied, allowed, and restored correctly as consent is revoked and regranted. In a live OpenAI RAG evaluation, **17 of 20** answerable questions were answered correctly (accuracy **= 0.85**), every answer carried a citation, all five unanswerable probes were safely refused, and mean latency was **1.67 s** (p95 **2.43 s**). Within this synthetic evaluation scope, the results show that separating voice processing, tool policy, clinical retrieval, and compliance logging yields a practical, auditable healthcare agent architecture.

**Index Terms**—Healthcare AI, voice agents, FHIR, retrieval-augmented generation, Model Context Protocol, consent, audit, microservices, Synthea.

---

## I. Introduction

### A. Background

Telephone and voice channels remain among the most accessible ways for patients to interact with health services, especially for people who struggle with app-centric portals, small screens, or low digital literacy. Large language models (LLMs) now make natural spoken dialogue practical, so a voice agent can answer questions about a patient's own medications, allergies, and conditions. Healthcare, however, is a high-stakes domain: any system that touches clinical records must protect privacy, respect patient consent, keep an audit trail, and avoid presenting invented facts as medical information. The World Health Organization's guidance on large multi-modal models for health emphasizes exactly these governance, accountability, and escalation requirements [WHO, 2024].

### B. Limitations of previous research

Previous research leaves three gaps. First, the most capable voice health systems, such as Polaris [Mukherjee et al., 2024], are proprietary: their safety engineering cannot be inspected, reproduced, or extended by the community. Second, the Model Context Protocol (MCP) [MCP Spec, 2025] standardizes how agents call tools, but it explicitly cannot enforce consent or authorization at the protocol layer; every implementor must design those controls, and MCP security surveys document the resulting threat surface [Hou et al., 2025]. Third, safety studies show that crisis and emergency detection degrades sharply on indirect, paraphrased language [Deng et al., 2025; Arnaiz-Rodriguez et al., 2026], yet published healthcare agent evaluations rarely stress-test this failure mode or release the data needed to reproduce it.

### C. Proposed approach and objectives

To address these gaps, we propose Zorba Health, an open-source microservice platform in which a voice agent can only reach clinical data through an MCP gateway that checks typed patient consents and writes append-only audit events before any tool runs. Retrieval-augmented generation (RAG) [Lewis et al., 2020] over FHIR-derived patient records supplies grounded, citation-bearing answers. The objective of this study is to answer one engineering question: how can a voice healthcare assistant be built so that tool use, record retrieval, and AI summarization are **consent-gated**, **auditable**, **grounded in patient-specific FHIR context**, and **reproducibly evaluable**? Our contributions are:

1. **Open-source voice healthcare architecture** — Go microservices, Next.js surfaces, gRPC, RabbitMQ, PostgreSQL (pgvector), Redis, LiveKit voice integration.
2. **Controlled MCP tool gateway** — agents invoke healthcare tools only through `mcp-server`, materializing the consent flows the MCP specification leaves to implementors.
3. **FHIR-compatible RAG pipeline** — ingest R4 bundles, chunk/embed, retrieve with filters/rerank, summarize over retrieved chunks only, return citations.
4. **Safety, consent, and audit framework** — `audit-service` with grant/revoke history and append-only events; voice-side emergency escalation.
5. **Reproducible synthetic evaluation package** — corpus, offline baselines, live RAG harness, consent-gate tests, and paraphrase safety stress test.

### D. Organization

The rest of this paper is organized as follows. Section II reviews related work and its limitations. Section III presents the proposed method. Section IV details data collection. Section V presents the experimental setup and results. Section VI discusses design implications and limitations. Section VII concludes.

---

## II. Related Work

**Voice and conversational health systems.** Relational agents and virtual nurses have long targeted low-literacy and telephone-accessible care [Bickmore et al., 2009]. Modern LLM medical systems encode substantial clinical knowledge [Singhal et al., 2023; Nori et al., 2023], but many demos treat the dialogue model as the system of record. Polaris [Mukherjee et al., 2024] is the closest published voice healthcare agent: a proprietary multi-agent constellation evaluated by over 1,100 U.S. licensed nurses and 130 physicians, reporting aggregate parity with human nurses on conversational and safety axes. Zorba Health differs on three axes: (i) it is fully open-source with reproducible synthetic evaluation; (ii) it centers consent-gated MCP mediation and append-only audit rather than constellation training; and (iii) it makes no clinical-parity claims.

**FHIR and synthetic EHR.** HL7 FHIR R4 is the dominant exchange model for interoperable health apps [Mandl et al., 2015; HL7 FHIR R4]. Synthea generates longitudinal synthetic patients widely used for methods research without PHI [Walonoski et al., 2018]. MedAgentBench [Jiang et al., 2025] further provides a FHIR-compliant interactive EHR environment (100 patients, 300 clinician-authored tasks) for benchmarking medical LLM agents—an evaluation target complementary to our synthetic corpus.

**RAG for clinical QA.** Retrieval-augmented generation grounds LLMs in external evidence [Lewis et al., 2020]. Medical RAG benchmarks and automatic faithfulness metrics (e.g., RAGAS) quantify hallucination risk [Es et al., 2023; Xiong et al., 2024]. Almanac [Zakka et al., 2024] shows clinician-panel preference for guideline-grounded RAG over ungrounded chatbots on factuality and adversarial safety. FHIR-RAG-MEDS [Kabak et al., 2026] integrates HL7 FHIR with RAG for personalized decision support over clinical guidelines. Our pipeline likewise enforces patient-scoped vector search and consent checks before summarization, and returns citations with answers, but targets patient-record QA rather than guideline retrieval.

**Agent tool use and MCP.** Tool-using LLMs (ReAct, Toolformer) show that models can call external actions [Yao et al., 2023; Schick et al., 2023]. AgentClinic [Schmidgall et al., 2024] and HealthBench [Arora et al., 2025] stress that interactive, open-ended clinical evaluation is harder than static QA; HealthBench further includes physician rubrics spanning emergency referral and safety behaviors. The Model Context Protocol (MCP) standardizes tool exposure to agents [MCP Spec, 2025]. Early healthcare MCP prototypes document privacy and compliance layers [Shehab, 2025; ElSayed et al., 2025], while Hou et al. [2025] systematize MCP threat scenarios (tool poisoning, unauthorized access, installer spoofing). The MCP specification itself states that the protocol cannot enforce consent or authorization at the protocol layer and that implementors SHOULD build robust consent flows [MCP Spec, 2025]. We treat MCP as a mandatory mediation layer and materialize consent and audit as first-class services around it.

**Consent, audit, and crisis escalation.** HIPAA [1996] and WHO guidance on large multi-modal models for health [WHO, 2024] motivate purpose limitation, access control, accountable logging, and careful escalation paths. Break-glass emergency patterns further require explicit detection of crisis language. PsyCrisisBench [Deng et al., 2025] shows LLMs reaching F1 ≈ 0.88 on suicidal-ideation detection in psychological-hotline transcripts, while Arnaiz-Rodriguez et al. [2026] find that all evaluated models still struggle with indirect crisis signals. These results frame our paraphrase-F1 collapse (0.29) as a known failure mode of surface-form safety rules and motivate LLM- or embedding-based escalation as future work. Zorba Health materializes consent and audit as a dedicated service with typed consents checked before record retrieval, summarization, and location tools.

**Limitations of prior work.** Across these studies, three limitations persist. Open, end-to-end voice healthcare systems with enforceable consent do not exist: Polaris is closed, and the MCP healthcare prototypes stop short of a running platform with audit guarantees. Consent and audit are treated as policy discussion rather than as tested system components, despite HIPAA and WHO guidance demanding both. Finally, safety evaluations rarely publish paraphrase stress sets, so brittleness like that documented by PsyCrisisBench remains invisible in system papers. These limitations motivate an open architecture whose consent gating, audit logging, and safety behavior are all directly measurable, which is what Zorba Health provides.

---

## III. Proposed Method

A new approach is necessary because none of the systems above combines, in one inspectable platform, the four properties a clinical voice agent needs: mediated tool access, enforceable per-patient consent, append-only auditing, and grounded answers with citations. Bolting a consent check onto a monolithic agent does not work, because the agent process itself still holds database credentials; the design goal is therefore architectural separation, so that the voice runtime is physically unable to touch clinical data except through a policy-checking gateway. This directly targets the research problem from Section I: consent gating, auditability, and grounding become properties of the system topology rather than of prompt engineering.

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

Voice sessions do **not** query FHIR stores directly. The agent requests tools via MCP; MCP calls gRPC services and records audit outcomes. Figure 1 of the LaTeX camera-ready shows the topology: the voice runtime (blue) can reach clinical data (green/teal) only through the MCP gateway (orange), which checks consent with the audit service (red) first. Figure 2 details one record question as a numbered sequence: (1) the agent issues a tool call; (2) the gateway asks the audit service whether `HEALTH_RECORD_ACCESS` is active; (3) if allowed, the query goes to the RAG service; (4) a patient-scoped vector search runs; (5) an audit event records what was retrieved. A missing consent stops the flow at step 2, and the denial itself is audited.

### B. Consent-aware RAG

Implemented in `services/health-records-service/internal/rag`:

1. Validate patient ID and query.  
2. Check `HEALTH_RECORD_ACCESS`; on denial, audit and abort.  
3. Embed query (default `text-embedding-3-small`).  
4. Patient-scoped vector search with over-fetch, metadata filter, light rerank.  
5. Optional LLM summarization **only** over retrieved chunk texts.  
6. Audit `HEALTH_RECORD_SEARCHED` / `HEALTH_RECORD_SUMMARIZED`.  
7. Return answer + citations.

The present boundary is asymmetric: `HEALTH_RECORD_ACCESS` is enforced before retrieval, but `THIRD_PARTY_MODEL_PROCESSING` is **not** yet checked before PHI-derived text is sent to external embedding or chat providers. The planned gate checks both consent types before any patient-derived text leaves the local trust boundary, denies external processing when the third-party grant is absent, and audits the consent decision, purpose, provider, and outcome.

### C. Safety escalation

`voice-agent-service` applies a deterministic phrase classifier (`safety.py`) for emergency-like utterances, triggers hospital-visible escalation, and emits `EMERGENCY_ESCALATION_TRIGGERED` audit events. This rule is a transparent high-precision baseline; Section V shows it is not deployment-ready for paraphrased emergencies. The planned mitigation keeps exact-phrase matching as a first layer, adds a semantic or embedding/LLM detector for indirect language, and escalates or asks a clarifying question when confidence is uncertain. That hybrid detector must still be evaluated on the released paraphrase set and clinician-reviewed cases before clinical use; it is not claimed as implemented here.

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

All services run locally via Docker Compose; scoring uses Python scripts in `scripts/evaluation/` and a Go harness that drives the production RAG code path against a local PostgreSQL/pgvector instance and the OpenAI embedding and chat APIs.

1. Existence oracle + type-aware lexical Recall@3 (does the expected resource appear in the top k=3 matches?)  
2. Live OpenAI RAG harness (`cmd/eval-live-rag`) with **answer-string-required** audit (`audit_live_rag_results.py`): an answer counts as correct only if the expected fact appears in the answer text itself; a citation alone is not enough  
3. Consent stub e2e mirroring `rag.Pipeline` deny/allow (`score_consent_e2e.py`)  
4. Exact + paraphrase safety scoring (`score_safety_paraphrase.py`), reported as precision P, recall R, and F1 = 2PR / (P + R), where a true positive is an emergency utterance that triggers escalation  
5. Go unit tests for FHIR validate + RAG mocks  

### C. Results

#### Table II — Lexical retrieval proxy

| Metric | Value |
| --- | ---: |
| Existence oracle recall | **1.00** |
| Type-aware Recall@3 | **1.00** |

On this corpus, retrieval never fails to surface the right resource, so any answer errors come from generation, not search.

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

Exact-set perfection is a regression sanity check. Paraphrase recall collapse shows the current classifier is **not** clinically robust under rewording: a patient who says "my chest feels like it's being squeezed" instead of "chest pain" is missed. Precision stays 1.00 throughout — the rule never over-triggers — so the failure mode is silence, not noise. The LaTeX camera-ready adds a color bar chart (Fig. 4) of these splits. This mirrors the indirect-signal failures reported by PsyCrisisBench [Deng et al., 2025] and Arnaiz-Rodriguez et al. [2026], whose LLM-based classifiers reach F1 ≈ 0.88 on explicit ideation; upgrading from phrase rules to LLM or embedding classifiers is the clear next step, and our released paraphrase set makes that comparison reproducible.

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

## VI. Discussion and Limitations

### A. Design implications

MCP + audit as first-class services create a single policy enforcement point shared by voice and web. Relative to Hou et al.'s MCP threat taxonomy [Hou et al., 2025], the gateway and audit path mitigate unauthorized tool invocation (deny-by-default consent gates), weak accountability (append-only event log), and unmediated PHI access (no direct store queries from the voice runtime). They do **not** yet address installer spoofing, malicious third-party MCP server listings, or cryptographic manifest signing for tool descriptors—orthogonal hardening left to deployment and future work. Relative to Polaris [Mukherjee et al., 2024], we trade proprietary constellation training and clinician-parity claims for an auditable open architecture and reproducible synthetic metrics.

### B. Scope of the evaluation

The reported metrics validate engineering behavior on a released synthetic FHIR corpus; they do **not** establish clinical efficacy, real-world crisis safety, or generalization to production electronic health records. Synthetic FHIR under-represents messy clinical narratives, the live-RAG gold set is modest (25 QA items), and locally measured latency understates telephony round trips. A staged external-validation path is therefore required: MedAgentBench-style interactive tasks [Jiang et al., 2025], clinician-rated groundedness and safety review, and only then governed evaluation on de-identified or real clinical data under IRB and privacy approval.

### C. Prioritized camera-ready limitations

1. **Paraphrase-robust escalation.** The phrase-based escalator remains high-precision but low-recall on paraphrases (F1 0.29), so a hybrid exact-phrase plus semantic/LLM detector with fail-safe clarification must be evaluated before deployment [Deng et al., 2025; Arnaiz-Rodriguez et al., 2026].
2. **Third-party model consent.** `THIRD_PARTY_MODEL_PROCESSING` is not yet enforced before external embedding or chat calls; the next implementation milestone is a deny-by-default gate with audited purpose, provider, and outcome.
3. **Standards interoperability.** Typed consents should be mapped to FHIR Consent [HL7 FHIR Consent] and composed with SMART scopes [SMART App Launch] for EHR interoperability.

### D. Threats to validity

Synthetic FHIR under-represents messy notes. Local latency underestimates telephony RTT. Stub consent e2e validates pipeline contract, not full HTTP portal auth.

---

## VII. Conclusion

This paper asked how a voice healthcare assistant can be engineered so that tool use, record retrieval, and AI summarization are consent-gated, auditable, grounded in patient-specific FHIR context, and reproducibly evaluable. Zorba Health answers with architecture rather than prompts: an open-source microservice platform in which an MCP gateway and an append-only audit/consent service stand between the voice agent and all clinical data, and a FHIR-grounded RAG pipeline returns cited answers. On the released synthetic corpus, retrieval is perfect (**Recall@3 = 1.00**), live RAG reaches **0.85** answer accuracy with full citation coverage and safe refusals, and consent deny/allow/regrant for health-record access is enforced end-to-end. An honest paraphrase stress test also exposes the current limit of rule-based escalation (**F1 1.00 → 0.29**), and third-party model consent remains an open enforcement gap before external LLM providers. Within this synthetic scope, the identified problem — uncontrolled, unauditable agent access to clinical context — is addressable with system topology, and the open evaluation package lets others verify or beat these numbers. Future work prioritizes hybrid paraphrase-robust escalation, third-party consent gates inside RAG, FHIR Consent / SMART alignment, MedAgentBench- and HealthBench-style interactive evaluation, clinician review, and governed studies beyond synthetic data.

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
- [x] GAISS conference framing (six-page limit) + ethics/COI  
- [x] Camera-ready responses to synthetic-data, paraphrase-safety, and third-party consent comments  
- [x] Full LaTeX + figures + PDF  
- [x] Reproducibility freeze notes  
- [ ] Optional: institutional affiliation upgrade  
- [ ] Optional: clinician spot-check (future work / journal extension)

## Appendix B — Cover letter blurb

We submit a systems paper on Zorba Health, an open-source voice healthcare platform that mediates LLM tool use through a consent- and audit-aware MCP gateway and answers patient questions with FHIR-compatible RAG and citations. We release a synthetic evaluation corpus (16 patients, 646 resources) and report: lexical Recall@3 = 1.00; live OpenAI RAG answer-string accuracy = 0.85 with citation presence = 1.00 and safe refusal on unanswerable LDL probes (mean 1.67 s); consent deny/allow/regrant pass for health-record access; and an honest paraphrase stress test where emergency-escalation F1 falls from 1.00 (exact) to 0.29 (paraphrase). The camera-ready revision clarifies the synthetic evaluation scope, the planned hybrid paraphrase-robust escalation path, and the remaining third-party model-consent gate. The contribution is an auditable architecture and reproducible evaluation path, not a claim of clinical efficacy.
