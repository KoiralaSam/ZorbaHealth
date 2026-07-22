#!/usr/bin/env python3
"""Offline retrieval baselines over curated FHIR bundles + gold QA.

Metrics:
1) existence_oracle — expected fact string present anywhere in patient chunks
2) type_aware_lexical@3 — prefer FHIR resource types hinted by question_type,
   then rank by token overlap
"""

from __future__ import annotations

import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
EVAL = ROOT / "examples" / "evaluation-data"
GOLD = EVAL / "gold" / "gold_qa.jsonl"

TYPE_HINT = {
    "condition": "Condition",
    "medication": "MedicationRequest",
    "allergy": "AllergyIntolerance",
    "observation": "Observation",
}


def chunk_records(bundle: dict) -> list[dict]:
    out = []
    for entry in bundle.get("entry", []):
        res = entry.get("resource") or {}
        rt = res.get("resourceType", "")
        text_parts = [rt]
        if rt == "Condition":
            text_parts.append((res.get("code") or {}).get("text", ""))
        elif rt == "MedicationRequest":
            text_parts.append(
                (res.get("medicationCodeableConcept") or {}).get("text", "")
            )
        elif rt == "AllergyIntolerance":
            text_parts.append((res.get("code") or {}).get("text", ""))
        elif rt == "Observation":
            code = (res.get("code") or {}).get("text", "")
            vq = res.get("valueQuantity") or {}
            text_parts.append(f"{code} {vq.get('value', '')} {vq.get('unit', '')}")
        elif rt == "Patient":
            n = (res.get("name") or [{}])[0]
            text_parts.append(
                f"{' '.join(n.get('given') or [])} {n.get('family', '')}"
            )
        text = " ".join(str(x) for x in text_parts if x).strip()
        if text:
            out.append({"resource_type": rt, "text": text})
    return out


def tokenize(s: str) -> set[str]:
    return set(re.findall(r"[a-z0-9]+", s.lower()))


def rank(query: str, chunks: list[dict], qtype: str, k: int = 3) -> list[dict]:
    q = tokenize(query)
    hint = TYPE_HINT.get(qtype)

    def score(c: dict) -> float:
        base = len(q & tokenize(c["text"])) / max(1, len(q))
        boost = 1.0 if hint and c["resource_type"] == hint else 0.0
        return boost + base

    return sorted(chunks, key=score, reverse=True)[:k]


def contains_expected(text: str, needles) -> bool:
    if isinstance(needles, str):
        needles = [needles]
    low = text.lower()
    return any(str(n).lower() in low for n in needles if n)


def main() -> None:
    rows = [
        json.loads(line)
        for line in GOLD.read_text(encoding="utf-8").splitlines()
        if line.strip()
    ]
    oracle_hit = 0
    type_hit = 0
    answerable = 0
    unanswerable = 0
    unanswerable_ok = 0
    details = []

    for r in rows:
        bundle = json.loads((ROOT / r["bundle_file"]).read_text(encoding="utf-8"))
        chunks = chunk_records(bundle)
        all_text = "\n".join(c["text"] for c in chunks)

        if r["expected_behavior"] != "grounded_answer":
            unanswerable += 1
            top = rank(r["question"], chunks, r["question_type"], k=3)
            joined = "\n".join(c["text"] for c in top).lower()
            ok = "ldl" not in joined
            unanswerable_ok += int(ok)
            details.append({"id": r["id"], "type": "unanswerable", "ok": ok})
            continue

        answerable += 1
        needles = r.get("expected_answer_contains") or []
        oracle = contains_expected(all_text, needles)
        oracle_hit += int(oracle)
        top = rank(r["question"], chunks, r["question_type"], k=3)
        joined = "\n".join(c["text"] for c in top)
        ok = contains_expected(joined, needles)
        type_hit += int(ok)
        details.append(
            {
                "id": r["id"],
                "type": r["question_type"],
                "oracle_ok": oracle,
                "type_aware_ok": ok,
                "top1": top[0]["text"] if top else "",
            }
        )

    summary = {
        "answerable_n": answerable,
        "existence_oracle_recall": round(oracle_hit / answerable, 4) if answerable else 0.0,
        "type_aware_recall_at_3": round(type_hit / answerable, 4) if answerable else 0.0,
        "answerable_oracle_hits": oracle_hit,
        "answerable_type_aware_hits": type_hit,
        "unanswerable_n": unanswerable,
        "unanswerable_no_ldl_leak_rate": round(unanswerable_ok / unanswerable, 4)
        if unanswerable
        else 0.0,
        "method": "existence_oracle + type_aware_lexical@3",
    }
    out = EVAL / "gold" / "keyword_retrieval_results.json"
    out.write_text(json.dumps({"summary": summary, "items": details}, indent=2) + "\n")
    print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()
