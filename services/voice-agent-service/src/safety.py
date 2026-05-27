from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class SafetyDecision:
    should_escalate: bool
    reason: str = ""
    guidance: str = ""


_EMERGENCY_PATTERNS: dict[str, tuple[str, str]] = {
    "chest pain": (
        "chest pain",
        "This may be an emergency. Please hang up and call 9-1-1 now, or go to the nearest emergency department immediately.",
    ),
    "shortness of breath": (
        "severe breathing difficulty",
        "This may be an emergency. Please call 9-1-1 now or seek emergency care immediately.",
    ),
    "can not breathe": (
        "severe breathing difficulty",
        "This may be an emergency. Please call 9-1-1 now or seek emergency care immediately.",
    ),
    "stroke": (
        "stroke symptoms",
        "Stroke symptoms need immediate care. Please call 9-1-1 now.",
    ),
    "suicidal": (
        "suicidal ideation",
        "You deserve immediate help. Please call 9-8-8 now if you are in the U.S., or call 9-1-1 if you are in immediate danger.",
    ),
    "overdose": (
        "medication overdose",
        "An overdose can be life-threatening. Please call 9-1-1 now or go to the nearest emergency department immediately.",
    ),
    "uncontrolled bleeding": (
        "uncontrolled bleeding",
        "Heavy bleeding is an emergency. Please call 9-1-1 now or get urgent emergency help immediately.",
    ),
    "loss of consciousness": (
        "loss of consciousness",
        "Loss of consciousness is an emergency. Please call 9-1-1 now.",
    ),
    "seizure": (
        "seizure",
        "A seizure can require emergency care. Please call 9-1-1 now if the seizure is ongoing, repeated, or the person is not recovering.",
    ),
}


def evaluate_text(text: str) -> SafetyDecision:
    normalized = " ".join(text.lower().split())
    for phrase, (reason, guidance) in _EMERGENCY_PATTERNS.items():
        if phrase in normalized:
            return SafetyDecision(should_escalate=True, reason=reason, guidance=guidance)
    return SafetyDecision(should_escalate=False)
