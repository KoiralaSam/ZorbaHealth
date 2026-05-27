from __future__ import annotations

import asyncio
import json
import logging

from livekit.agents import RunContext, function_tool

from tools.mcp_client import MCPClient, MCPToolError
from userdata import SessionUserData

logger = logging.getLogger("zorba.tools")


def _client(context: RunContext[SessionUserData]) -> MCPClient:
    client = context.userdata.mcp_client
    if client is None:
        raise MCPToolError("MCP client is not configured for this session")
    return client


async def notify_call_lifecycle_for_session(
    ud: SessionUserData,
    mcp_client: MCPClient,
    event_type: str,
) -> None:
    """Publish call.started / call.ended for location-service → patient app WS."""
    if not ud.is_verified or not ud.verified_patient_id or not ud.session_id:
        return
    try:
        await mcp_client.call_tool(
            "notify_call_lifecycle",
            {
                "eventType": event_type,
                "sessionID": ud.session_id,
                "patientID": ud.verified_patient_id,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning(
            "notify_call_lifecycle %s failed session=%s: %s",
            event_type,
            ud.session_id,
            exc,
        )


async def _notify_call_lifecycle(
    context: RunContext[SessionUserData],
    event_type: str,
) -> None:
    ud = context.userdata
    client = context.userdata.mcp_client
    if client is None:
        return
    await notify_call_lifecycle_for_session(ud, client, event_type)


def _apply_verified_session(ud: SessionUserData, patient_id: str, access_token: str) -> None:
    if not patient_id:
        raise MCPToolError("verified patient session is incomplete")
    if ud.session_auth is None:
        raise MCPToolError("session auth is not configured for this voice session")
    # Auth-service JWTs use the login session UUID; location is keyed by LiveKit room id.
    # Re-mint a voice-scoped patient token so MCP sessionID checks match this call.
    _ = access_token
    voice_token = ud.session_auth.mint_patient_token(
        patient_id=patient_id,
        session_id=ud.session_id,
        scopes=["location:read", "records:read"],
    )
    logger.info(
        "verified patient session established session=%s patient_id=%s",
        ud.session_id,
        patient_id,
    )
    ud.upgrade(patient_id, voice_token)


@function_tool
async def translate(
    context: RunContext[SessionUserData],
    text: str,
    target_lang: str,
    source_lang: str = "",
) -> str:
    """Translate text into another language.

    Args:
        text: The text to translate.
        target_lang: Target ISO 639-1 language code.
        source_lang: Optional source ISO 639-1 language code.
    """
    ud = context.userdata
    try:
        return await _client(context).call_tool(
            "translate",
            {
                "text": text,
                "targetLang": target_lang,
                "sourceLang": source_lang,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("translate failed session=%s: %s", ud.session_id, exc)
        return "I was unable to translate that right now."


@function_tool
async def get_location(context: RunContext[SessionUserData]) -> str:
    """Get the caller's current location."""
    ud = context.userdata
    client = _client(context)
    payload = {
        "sessionID": ud.session_id,
        "_auth": ud.active_token,
    }
    # Portal GPS can take a few seconds after start_location (browser prompt).
    for attempt in range(4):
        try:
            raw = await client.call_tool("get_location", payload)
            if '"available":false' not in raw and "no_location" not in raw:
                return raw
            if attempt < 3:
                await asyncio.sleep(3)
        except MCPToolError as exc:
            if attempt >= 3:
                logger.warning("get_location failed session=%s: %s", ud.session_id, exc)
                return "I was unable to determine your location right now."
            await asyncio.sleep(3)
    return (
        "No GPS fix yet. Ask the patient to allow location in the browser on the Zorba "
        "patient portal (same account), then try again."
    )


@function_tool
async def find_nearest_hospital(
    context: RunContext[SessionUserData],
    lat: float,
    lng: float,
    place_type: str = "hospital",
) -> str:
    """Find the nearest hospital, urgent care clinic, or pharmacy.

    Args:
        lat: Latitude from get_location.
        lng: Longitude from get_location.
        place_type: One of hospital, urgent_care, or pharmacy.
    """
    ud = context.userdata
    try:
        return await _client(context).call_tool(
            "find_nearest_hospital",
            {
                "lat": lat,
                "lng": lng,
                "placeType": place_type,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("find_nearest_hospital failed session=%s: %s", ud.session_id, exc)
        return "I was unable to find nearby facilities right now. Please call 9-1-1 if this is an emergency."


@function_tool
async def search_health_records(
    context: RunContext[SessionUserData],
    query: str,
    top_k: int = 5,
) -> str:
    """Search verified patient health records.

    Only call this after identity verification has provided a patient token.

    Args:
        query: Health-record search query.
        top_k: Maximum number of matching chunks to return.
    """
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before I can access personal health records."
    try:
        return await _client(context).call_tool(
            "search_health_records",
            {
                "query": query,
                "topK": top_k,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("search_health_records failed session=%s: %s", ud.session_id, exc)
        return "I was unable to search your health records right now."


@function_tool
async def answer_health_question(
    context: RunContext[SessionUserData],
    question: str,
    top_k: int = 5,
) -> str:
    """Answer a verified patient's question using grounded health records."""
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before I can answer questions about your records."
    if not question.strip():
        return "Please tell me what you want to know from your records."
    try:
        raw = await _client(context).call_tool(
            "answer_health_question",
            {
                "question": question.strip(),
                "topK": top_k,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("answer_health_question failed session=%s: %s", ud.session_id, exc)
        return "I was unable to answer that from your health records right now."

    try:
        payload = json.loads(raw)
    except json.JSONDecodeError:
        return raw

    answer = str(payload.get("answer") or "").strip()
    return answer or raw


@function_tool
async def lookup_patient_by_phone(context: RunContext[SessionUserData], phone_number: str = "") -> str:
    """Check whether the caller phone already belongs to an existing patient."""
    ud = context.userdata
    phone = phone_number.strip() or ud.caller_phone
    if not phone:
        return "I need a phone number before I can look up your account."
    try:
        return await _client(context).call_tool(
            "lookup_patient_by_phone",
            {
                "phoneNumber": phone,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("lookup_patient_by_phone failed session=%s: %s", ud.session_id, exc)
        return "I was unable to look up that phone number right now."


@function_tool
async def start_existing_phone_verification(context: RunContext[SessionUserData], phone_number: str = "") -> str:
    """Send an OTP to an existing patient on the caller phone number."""
    ud = context.userdata
    phone = phone_number.strip() or ud.caller_phone
    if not phone:
        return "I need a phone number before I can send a verification code."
    try:
        ud.verification_mode = "existing"
        ud.verification_state = "existing_patient_otp_pending"
        return await _client(context).call_tool(
            "start_existing_phone_verification",
            {
                "phoneNumber": phone,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("start_existing_phone_verification failed session=%s: %s", ud.session_id, exc)
        return "I was unable to send a verification code right now."


@function_tool
async def verify_existing_phone_otp(context: RunContext[SessionUserData], otp: str, phone_number: str = "") -> str:
    """Verify an existing patient by SMS OTP and upgrade the session token."""
    ud = context.userdata
    phone = phone_number.strip() or ud.caller_phone
    if not phone:
        return "I need a phone number before I can verify that code."
    if not otp.strip():
        return "Please provide the verification code."
    try:
        raw = await _client(context).call_tool(
            "verify_existing_phone_otp",
            {
                "phoneNumber": phone,
                "otp": otp.strip(),
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("verify_existing_phone_otp failed session=%s: %s", ud.session_id, exc)
        return "That verification code did not work."

    try:
        payload = json.loads(raw)
    except json.JSONDecodeError:
        return raw

    patient_id = str(payload.get("patientID") or "").strip()
    access_token = str(payload.get("accessToken") or "").strip()
    if patient_id and access_token:
        _apply_verified_session(context.session.userdata, patient_id, access_token)
        await _notify_call_lifecycle(context, "call.started")
    return str(payload.get("message") or raw)


@function_tool
async def log_escalation(
    context: RunContext[SessionUserData],
    reason: str,
    severity: str = "high",
) -> str:
    """Record an emergency escalation for the current caller session.

    Args:
        reason: Short emergency reason such as chest pain or stroke symptoms.
        severity: Escalation severity label.
    """
    ud = context.userdata
    try:
        return await _client(context).call_tool(
            "log_escalation",
            {
                "sessionID": ud.session_id,
                "patientID": ud.verified_patient_id or "",
                "callerPhone": ud.caller_phone,
                "reason": reason,
                "severity": severity,
                "transferRequested": ud.transfer_requested,
                "transferTarget": ud.transfer_target,
                "alertPhoneNumbers": ud.alert_phone_numbers or [],
                "transcriptExcerpt": ud.last_user_transcript,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("log_escalation failed session=%s: %s", ud.session_id, exc)
        return "Emergency escalation recording failed."


ALL_TOOLS = [
    translate,
    get_location,
    find_nearest_hospital,
    lookup_patient_by_phone,
    start_existing_phone_verification,
    verify_existing_phone_otp,
    search_health_records,
    answer_health_question,
    log_escalation,
]
