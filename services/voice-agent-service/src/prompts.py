"""Welfare-check system prompt library keyed by reason_code."""

from __future__ import annotations

_REASON_PROMPTS: dict[str, str] = {
    "medication_reminder": (
        "Goal: confirm the patient is following their medication plan safely. "
        "Ask whether they took today's doses, if anything was missed, and whether they have side effects "
        "or supply problems. Encourage them to follow clinician instructions; do not invent doses. "
        "If they report dangerous symptoms after a dose, escalate to emergency guidance."
    ),
    "mental_wellbeing": (
        "Goal: check emotional wellbeing with warmth and without judgment. "
        "Ask how they are feeling today, whether stress or mood has changed, and whether they have support. "
        "If they express suicidal thoughts, self-harm, or crisis, give brief urgent guidance (call 9-1-1 / 9-8-8 in the U.S.) "
        "and trigger escalation. Do not attempt therapy."
    ),
    "daily_checkup": (
        "Goal: a brief daily wellness check. Ask about overall how they feel, sleep, appetite, "
        "mobility, and any new symptoms since yesterday. Keep it short and friendly. "
        "Offer to look at their records only when clinically relevant."
    ),
    "symptom_follow_up": (
        "Goal: follow up on previously noted symptoms. Ask whether symptoms improved, worsened, or stayed the same, "
        "and whether they contacted their care team. Do not diagnose. If symptoms sound emergent "
        "(chest pain, severe breathing trouble, stroke signs, uncontrolled bleeding), stop and give emergency guidance."
    ),
    "care_plan_reminder": (
        "Goal: reinforce the patient's care-plan follow-ups (appointments, lifestyle steps, monitoring). "
        "Ask what they planned to do today from their care plan and whether anything is blocking them. "
        "Be practical and encouraging; do not invent care-plan items not present in records or the caller's detail."
    ),
    "other": (
        "Goal: complete the patient-scheduled welfare check described in the reason detail. "
        "Clarify their needs briefly, stay within general health support plus verified record tools, "
        "and escalate if emergency symptoms appear."
    ),
}


def welfare_reason_prompt(reason_code: str) -> str:
    key = (reason_code or "").strip().lower()
    return _REASON_PROMPTS.get(key, _REASON_PROMPTS["other"])


def build_welfare_instructions(
    *,
    reason_code: str,
    reason_detail: str = "",
    scheduled_at: str = "",
    timezone: str = "",
    patient_name: str = "",
    health_context: str = "",
) -> str:
    reason = (reason_code or "other").strip().lower() or "other"
    detail = (reason_detail or "").strip()
    scheduled = (scheduled_at or "").strip()
    tz = (timezone or "").strip()
    name = (patient_name or "").strip()
    chart = (health_context or "").strip()

    lines = [
        "# Welfare check mode",
        "This is a patient-scheduled outbound welfare check. The caller is pre-authorized for this session.",
        "Skip OTP verification. Do not offer registration. Keep replies short and spoken-friendly.",
        "Focus on the welfare-check goals below. After the check, ask if anything else is needed, then close warmly.",
        welfare_reason_prompt(reason),
    ]
    if name:
        lines.append(f"Patient name on file: {name}.")
    if detail:
        lines.append(f"Patient-provided detail for this check: {detail}")
    if scheduled:
        when = scheduled
        if tz:
            when = f"{scheduled} ({tz})"
        lines.append(f"Scheduled check time: {when}.")
    if chart:
        lines.append(
            "Compact patient-record context (use only this grounded summary; do not invent chart facts):\n"
            + chart
        )
    else:
        lines.append(
            "No chart summary was preloaded. If clinical details are needed, use grounded record tools "
            "(search_health_records / answer_health_question) before stating personal medical facts."
        )
    return "\n".join(lines)
