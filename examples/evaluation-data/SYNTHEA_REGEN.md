# Regenerating Synthea raw exports

Raw Synthea patient JSON files are large (~2–4 MB each). This repo keeps
**slimmed** bundles under `fhir-bundles/synthea-*.json` (supported FHIR types only).

```bash
mkdir -p /tmp/synthea && cd /tmp/synthea
curl -fsSL -o synthea.jar \
  https://github.com/synthetichealth/synthea/releases/download/master-branch-latest/synthea-with-dependencies.jar
java -jar synthea.jar -p 10 -s 1784242671144
python3 scripts/evaluation/build_eval_corpus.py
```

Seed used for the checked-in slim cohort: `1784242671144` (Massachusetts, n=10).
