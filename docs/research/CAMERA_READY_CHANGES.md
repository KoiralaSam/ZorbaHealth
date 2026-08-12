# GAISS Camera-Ready Change Record (Submission 7213)

Reviewer scores: 1 (weak accept), 2 (accept), 2 (accept). Teacher guidance: revise the conference paper for at least two comments; larger follow-ups may go to a journal extension.

Baseline PDF preserved as `/home/koiralas2/Samarpan_Final_Version_submitted_backup.pdf`.

## Reviewer concern → camera-ready response

| Concern | Where addressed | What changed |
| --- | --- | --- |
| Synthetic evaluation / need for realistic clinical validation | Abstract closing sentence; Section VI (scope paragraph); Section VII | Explicitly bound metrics to the released synthetic corpus; state that results validate engineering behavior, not clinical efficacy; add staged external-validation path (MedAgentBench → clinician review → IRB-governed real/de-identified data). |
| Paraphrase emergency escalation F1 = 0.29 | Section III.C; Section V results retained; Section VI prioritized limitations; Section VII | Keep reported F1 values unchanged; state classifier is not deployment-ready for paraphrases; describe planned hybrid mitigation (exact-phrase + semantic/LLM detector + fail-safe clarification) without claiming it is implemented. |
| Third-party model consent not fully enforced | Section III.B; Section V results wording; Section VI prioritized limitations; Section VII | Clarify asymmetric boundary (`HEALTH_RECORD_ACCESS` enforced, `THIRD_PARTY_MODEL_PROCESSING` not gated before external embedding/chat); describe planned deny-by-default audited gate. |

## Source fixes retained vs submitted PDF OCR/typesetting defects

- Removed duplicated Data Collection sentence present in the submitted PDF extract.
- Restored the complete experimental-setup description of the Go `eval-live-rag` harness and local PostgreSQL/pgvector path.
- Venue metadata corrected from stale “IEEE Access” labels to **GAISS** conference.

## Evidence-safety rules for this revision

- No new experimental numbers were invented.
- Frozen metrics (`Recall@3=1.00`, live RAG `0.85`, safety F1 exact/paraphrase/combined) remain unchanged.
- Future mitigations are labeled as planned, not completed.
