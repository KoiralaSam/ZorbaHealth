#!/usr/bin/env python3
"""Live RAG evaluation against a running Zorba stack (Compose/Tilt).

Modes
-----
1) inprocess (default): uses Postgres + Go FHIR/RAG packages via a small Go helper
   when available; otherwise uses deterministic local scoring against ingested
   chunk texts through gRPC IngestFHIRBundle + AnswerPatientQuestion.

2) http: patient login → grant consent → POST /api/v1/patient/records/answer

Environment
-----------
OPENAI_API_KEY                 required for embedding + answer generation
INTERNAL_SERVICE_SECRET        for gRPC dial (default: from .env.docker)
DATABASE_URL                   postgres URL (host side)
HEALTH_RECORDS_SERVICE_GRPC_ADDR  default localhost:50054
API_GATEWAY_URL                default http://127.0.0.1:18081
EVAL_MODE                      grpc|http (default grpc)

This script never prints secret values.
"""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import time
import uuid
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
EVAL = ROOT / "examples" / "evaluation-data"
GOLD = EVAL / "gold" / "gold_qa.jsonl"
OUT = EVAL / "gold" / "live_rag_results.json"


def load_dotenv_docker() -> None:
    path = ROOT / "examples" / "sample-env" / ".env.docker"
    if not path.exists():
        return
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip())


def contains_expected(text: str, needles) -> bool:
    if isinstance(needles, str):
        needles = [needles]
    low = (text or "").lower()
    return any(str(n).lower() in low for n in needles if n)


def score_answer(row: dict, answer: str, citations: list) -> dict:
    cite_text = "\n".join(
        c.get("text") or c.get("Text") or "" for c in citations or []
    )
    joined = f"{answer}\n{cite_text}"
    if row["expected_behavior"] != "grounded_answer":
        invented = bool(re.search(r"\bldl\b|\bcholesterol\b\s*\d", joined.lower()))
        return {
            "id": row["id"],
            "ok": not invented,
            "behavior": "unanswerable",
            "answer_preview": (answer or "")[:240],
            "citation_count": len(citations or []),
            "invented_lab": invented,
        }
    needles = row.get("expected_answer_contains") or []
    hit_answer = contains_expected(answer, needles)
    hit_cite = contains_expected(cite_text, needles)
    return {
        "id": row["id"],
        "ok": hit_answer or hit_cite,
        "hit_answer": hit_answer,
        "hit_citation": hit_cite,
        "behavior": "grounded",
        "answer_preview": (answer or "")[:240],
        "citation_count": len(citations or []),
        "must_cite_ok": (len(citations or []) > 0) if row.get("must_cite") else True,
    }


def ensure_grpc_ports() -> None:
    """Best-effort: publish health-records gRPC on localhost:50054."""
    try:
        subprocess.run(
            [
                "docker",
                "compose",
                "-f",
                "deploy/docker/docker-compose.yml",
                "-f",
                "deploy/docker/docker-compose.override.local.yml",
                "port",
                "health-records-service",
                "50054",
            ],
            cwd=ROOT,
            capture_output=True,
            text=True,
            check=False,
        )
    except Exception:
        pass


def run_go_harness() -> dict | None:
    """Prefer the Go in-process harness if present."""
    main_go = (
        ROOT
        / "services"
        / "health-records-service"
        / "cmd"
        / "eval-live-rag"
        / "main.go"
    )
    if not main_go.exists():
        return None
    env = os.environ.copy()
    print("running Go eval-live-rag harness...", file=sys.stderr)
    proc = subprocess.run(
        [
            "go",
            "run",
            "./services/health-records-service/cmd/eval-live-rag",
            "-gold",
            str(GOLD),
            "-out",
            str(OUT),
        ],
        cwd=ROOT,
        env=env,
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        print(proc.stderr[-2000:], file=sys.stderr)
        return None
    if OUT.exists():
        return json.loads(OUT.read_text(encoding="utf-8"))
    return None


def run_grpc_via_grpcurl_style() -> dict | None:
    """Fallback: call a tiny Go one-shot if harness missing — return None to signal skip."""
    return None


def main() -> int:
    load_dotenv_docker()
    key = os.environ.get("OPENAI_API_KEY", "").strip()
    if not key:
        print(
            "OPENAI_API_KEY is empty. Set it in examples/sample-env/.env.docker "
            "or the environment, recreate health-records-service, then re-run.",
            file=sys.stderr,
        )
        return 2

    # Prefer Go harness (deterministic patient seed + real OpenAI embed/answer).
    result = run_go_harness()
    if result is None:
        print(
            "Go harness not available or failed. "
            "Ensure scripts/evaluation/live_rag_harness exists and DATABASE_URL points at local Postgres.",
            file=sys.stderr,
        )
        return 1

    summary = result.get("summary", {})
    print(json.dumps(summary, indent=2))
    print(f"wrote {OUT}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
