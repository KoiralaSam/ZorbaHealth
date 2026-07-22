# Zorba Health Evaluation Dataset Card

## Summary

Synthetic evaluation corpus for the IEEE Access draft: FHIR patient bundles,
gold question–answer items, emergency escalation (exact + paraphrase), and consent flows.

| Item | Count |
| --- | ---: |
| Patients (catalog) | 16 |
| Curated gold patients | 5 |
| Synthea-slim patients | 10 |
| Gold QA items | 25 |
| Escalation exact | 22 (pos=10, neg=12) |
| Escalation paraphrase | 22 (pos=12, neg=10) |
| Consent scenarios | 8 |

## Headline metrics (paper freeze)

| Metric | Value |
| --- | ---: |
| Lexical Recall@3 | 1.00 |
| Live RAG answer-string accuracy | 0.85 (17/20) |
| Citation presence | 1.00 |
| Unanswerable safe refusal | 1.00 |
| Exact safety F1 | 1.00 |
| Paraphrase safety F1 | 0.29 |
| Consent e2e pass rate | 11/11 (1 known gap) |

## Ethics

Synthetic / demo data only. Do not mix production PHI into this directory.

## Regeneration

See `SYNTHEA_REGEN.md` and run `python3 scripts/evaluation/build_eval_corpus.py`.
Synthea seed: `1784242671144`.
