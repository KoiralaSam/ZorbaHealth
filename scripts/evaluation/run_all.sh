#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="${1:-$ROOT_DIR/scripts/evaluation/artifacts/$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$OUT_DIR"

echo "Running demo smoke checks..."
node "$ROOT_DIR/scripts/evaluation/demo-smoke.mjs" all > "$OUT_DIR/demo-smoke.json"

cat > "$OUT_DIR/summary.json" <<EOF
{
  "generated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "artifacts": {
    "demo_smoke": "demo-smoke.json"
  },
  "notes": [
    "Add k6 outputs under scripts/evaluation/load/ for latency and concurrency experiments.",
    "Add RAG precision/recall outputs under scripts/evaluation/datasets/ when labeled qrels are available."
  ]
}
EOF

cp "$ROOT_DIR/scripts/evaluation/report-template.md" "$OUT_DIR/report-template.md"
echo "Evaluation artifacts written to $OUT_DIR"
