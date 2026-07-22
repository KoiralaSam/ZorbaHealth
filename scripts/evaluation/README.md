# Evaluation Scripts

Journal-oriented evaluation tooling for Zorba Health (IEEE Access draft).

## Corpus

Built artifacts live in `examples/evaluation-data/` (see `DATASET_CARD.md`).

```bash
python3 scripts/evaluation/build_eval_corpus.py
python3 scripts/evaluation/score_keyword_retrieval.py
python3 scripts/evaluation/audit_live_rag_results.py
python3 scripts/evaluation/score_consent_e2e.py
python3 scripts/evaluation/score_safety_paraphrase.py

# Live OpenAI RAG (needs OPENAI_API_KEY + local Postgres)
export DATABASE_URL='postgres://healthai:healthai@localhost:5432/healthai?sslmode=disable'
go run ./services/health-records-service/cmd/eval-live-rag \
  -gold examples/evaluation-data/gold/gold_qa.jsonl \
  -out examples/evaluation-data/gold/live_rag_results.json
python3 scripts/evaluation/audit_live_rag_results.py
```

## Key outputs

| Script | Output |
| --- | --- |
| `score_keyword_retrieval.py` | `gold/keyword_retrieval_results.json` |
| `audit_live_rag_results.py` | `gold/live_rag_results_audited.json` |
| `score_consent_e2e.py` | `gold/consent_e2e_results.json` |
| `score_safety_paraphrase.py` | `gold/safety_paraphrase_results.json` + paraphrase jsonl |

See `docs/research/ieee-zorbahealth-draft.md` and `docs/research/latex/main.pdf`.
