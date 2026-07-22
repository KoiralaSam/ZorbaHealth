# Research / IEEE Access package

| Artifact | Path |
| --- | --- |
| Full markdown draft | [`ieee-zorbahealth-draft.md`](ieee-zorbahealth-draft.md) |
| Camera-ready PDF | [`latex/main.pdf`](latex/main.pdf) |
| IEEEtran LaTeX | [`latex/main.tex`](latex/main.tex) |
| BibTeX | [`latex/refs.bib`](latex/refs.bib) |
| Evaluation corpus | [`../../examples/evaluation-data/`](../../examples/evaluation-data/) |

## Venue

**IEEE Access** (systems contribution + open-source reproducibility).

## Reproduce tables

```bash
python3 scripts/evaluation/build_eval_corpus.py
python3 scripts/evaluation/score_keyword_retrieval.py          # Table II
python3 scripts/evaluation/audit_live_rag_results.py           # Table III (from prior live run)
python3 scripts/evaluation/score_consent_e2e.py                # Table V
python3 scripts/evaluation/score_safety_paraphrase.py          # Table IV

# Live RAG (needs OPENAI_API_KEY + local Postgres):
export DATABASE_URL='postgres://healthai:healthai@localhost:5432/healthai?sslmode=disable'
go run ./services/health-records-service/cmd/eval-live-rag \
  -gold examples/evaluation-data/gold/gold_qa.jsonl \
  -out examples/evaluation-data/gold/live_rag_results.json
python3 scripts/evaluation/audit_live_rag_results.py
```

## Compile PDF

```bash
cd docs/research/latex
tectonic main.tex
# opens: main.pdf
```

## Reproducibility freeze

- Synthea seed: `1784242671144`
- Draft freeze commit (at write time): `2d7429d0`
- Ethics: synthetic data only; no PHI
