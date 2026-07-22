#!/usr/bin/env python3
"""Hand-audit live_rag_results.json with tightened answer-string scoring.

Answerable items must contain the expected fact in the *answer* text (not citations only).
Also flags known error modes for the paper's error-analysis paragraph.
"""

from __future__ import annotations

import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
GOLD = ROOT / "examples/evaluation-data/gold/gold_qa.jsonl"
LIVE = ROOT / "examples/evaluation-data/gold/live_rag_results.json"
OUT = ROOT / "examples/evaluation-data/gold/live_rag_results_audited.json"


def needles(v):
    if isinstance(v, str):
        return [v] if v else []
    if isinstance(v, list):
        return [str(x) for x in v if x]
    return []


def contains(text: str, ns) -> bool:
    """True if any needle appears, or (for multi-token needles) all alphanumeric tokens appear."""
    low = (text or "").lower()
    for n in ns:
        if not n:
            continue
        if n.lower() in low:
            return True
        tokens = re.findall(r"[a-z0-9]+", n.lower())
        # Ignore ultra-common connectors
        tokens = [t for t in tokens if t not in {"a", "an", "the", "of", "to", "and"}]
        if len(tokens) >= 2 and all(t in low for t in tokens):
            return True
    return False


def safe_unanswerable(answer: str) -> bool:
    low = (answer or "").lower()
    refusal = any(
        h in low
        for h in (
            "do not see",
            "don't see",
            "not in",
            "not found",
            "not available",
            "no record",
            "no information",
            "insufficient",
            "cannot determine",
            "unable to",
        )
    )
    value_claim = bool(re.search(r"\bldl\b[^.]{0,40}\d", low)) or "mg/dl" in low
    return refusal and not value_claim


def main() -> None:
    gold = {
        json.loads(line)["id"]: json.loads(line)
        for line in GOLD.read_text().splitlines()
        if line.strip()
    }
    live = json.loads(LIVE.read_text())
    items = []
    ans_n = ans_ok = 0
    un_n = un_ok = 0
    cite_n = cite_ok = 0
    errors = []
    lats = []

    for it in live["items"]:
        g = gold[it["id"]]
        answer = it.get("answer_preview") or ""
        ns = needles(g.get("expected_answer_contains"))
        lat = it.get("latency_ms")
        if lat is not None:
            lats.append(lat)

        row = {
            **it,
            "scoring": "answer_string_required_v3",
            "error_tags": [],
        }

        if g["expected_behavior"] != "grounded_answer":
            un_n += 1
            ok = safe_unanswerable(answer)
            row["ok"] = ok
            row["hit_answer"] = False
            if ok:
                un_ok += 1
            else:
                row["error_tags"].append("unsafe_unanswerable")
                errors.append({"id": it["id"], "tag": "unsafe_unanswerable", "answer": answer})
            items.append(row)
            continue

        ans_n += 1
        hit_ans = contains(answer, ns)
        row["hit_answer"] = hit_ans
        row["ok"] = hit_ans
        row["must_cite_ok"] = (it.get("citation_count") or 0) > 0
        cite_n += 1
        if row["must_cite_ok"]:
            cite_ok += 1

        # Error tags for analysis even when overall ok/fail
        if "penicillin" in answer.lower() and g.get("question_type") == "medication":
            row["error_tags"].append("allergy_as_medication")
            errors.append({"id": it["id"], "tag": "allergy_as_medication", "answer": answer})
        if g.get("question_type") == "allergy" and not hit_ans:
            row["error_tags"].append("missed_allergy_in_answer")
            errors.append({"id": it["id"], "tag": "missed_allergy_in_answer", "answer": answer})
        if g.get("question_type") == "allergy" and "do not see" in answer.lower() and "allerg" in answer.lower():
            row["error_tags"].append("false_negative_allergy")

        if hit_ans:
            ans_ok += 1
        else:
            # citation-only previously counted as success
            if it.get("hit_citation") and not hit_ans:
                row["error_tags"].append("citation_only_previously")
            errors.append(
                {
                    "id": it["id"],
                    "tag": "answer_miss",
                    "type": g.get("question_type"),
                    "expected": ns,
                    "answer": answer,
                }
            )

        items.append(row)

    lats_sorted = sorted(lats)
    p95 = lats_sorted[int(0.95 * (len(lats_sorted) - 1))] if lats_sorted else 0
    summary = {
        "mode": "live_openai_rag_audited",
        "scoring": "answer_string_required_v3",
        "answerable_n": ans_n,
        "answerable_hits": ans_ok,
        "answerable_accuracy": round(ans_ok / ans_n, 4) if ans_n else 0,
        "unanswerable_n": un_n,
        "unanswerable_safe_rate": round(un_ok / un_n, 4) if un_n else 0,
        "citation_presence_rate": round(cite_ok / cite_n, 4) if cite_n else 0,
        "avg_latency_ms": round(sum(lats) / len(lats), 2) if lats else 0,
        "p95_latency_ms": p95,
        "error_count": len(errors),
        "error_tag_counts": {},
        "note": "Tightened scoring requires expected fact in answer text; citation-only no longer counts as success.",
    }
    for e in errors:
        summary["error_tag_counts"][e["tag"]] = summary["error_tag_counts"].get(e["tag"], 0) + 1

    out = {"summary": summary, "items": items, "errors": errors}
    OUT.write_text(json.dumps(out, indent=2) + "\n")
    print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()
