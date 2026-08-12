from __future__ import annotations

import asyncio
import json
import logging
import re
from datetime import datetime, timedelta, timezone
from zoneinfo import ZoneInfo

from livekit.agents import RunContext, ToolError, function_tool

from tools.mcp_client import MCPClient, MCPToolError
from userdata import SessionUserData

logger = logging.getLogger("zorba.tools")

_DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
_TIME_RE = re.compile(r"^\d{1,2}:\d{2}$")


def _visit_start_rfc3339(visit_date: str, visit_time_local: str, timezone_name: str) -> str:
    """Build UTC RFC3339 from caller-local calendar date and 24h clock time."""
    date_s = visit_date.strip()
    time_s = visit_time_local.strip().lower().replace(" ", "")
    tz_s = timezone_name.strip()
    if not _DATE_RE.match(date_s):
        raise MCPToolError("visit_date must be YYYY-MM-DD (confirm year and month with the caller).")
    if not _TIME_RE.match(time_s):
        raise MCPToolError("visit_time_local must be HH:MM in 24-hour local time (example: 14:30).")
    try:
        tz = ZoneInfo(tz_s)
    except Exception as exc:
        raise MCPToolError(f"Unknown timezone {tz_s!r}. Use an IANA name like America/New_York.") from exc
    hour_str, minute_str = time_s.split(":", 1)
    hour, minute = int(hour_str), int(minute_str)
    if hour > 23 or minute > 59:
        raise MCPToolError("visit_time_local is not a valid clock time.")
    year, month, day = (int(part) for part in date_s.split("-"))
    try:
        local = datetime(year, month, day, hour, minute, tzinfo=tz)
    except ValueError as exc:
        raise MCPToolError(f"Invalid visit date or time: {exc}") from exc
    utc = local.astimezone(timezone.utc)
    min_start = datetime.now(timezone.utc) + timedelta(minutes=10)
    if utc <= min_start:
        raise MCPToolError(
            "That visit time is not far enough in the future. "
            f"Server UTC now is {min_start.isoformat()}; "
            f"your entry parses to {utc.isoformat()} UTC ({local.isoformat()} local). "
            "Ask for a later date or time."
        )
    return utc.strftime("%Y-%m-%dT%H:%M:%SZ")


def _voice_caller_phone(ud: SessionUserData) -> str:
    """SIP voice sessions must use the provisioned caller phone, not LLM-supplied overrides."""
    if ud.caller_phone:
        return ud.caller_phone
    return ""


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
        caller_phone=ud.caller_phone,
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
async def lookup_patient_by_phone(context: RunContext[SessionUserData], phone_number: str | None = None) -> str:
    """Check whether the caller phone already belongs to an existing patient."""
    ud = context.userdata
    phone = _voice_caller_phone(ud) or (phone_number or "").strip()
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
async def start_existing_phone_verification(context: RunContext[SessionUserData], phone_number: str | None = None) -> str:
    """Send an OTP to an existing patient on the caller phone number."""
    from verification import begin_verification_flow, wait_for_sms_verification

    ud = context.userdata
    phone = _voice_caller_phone(ud) or (phone_number or "").strip()
    if not phone:
        return "I need a phone number before I can send a verification code."
    try:
        ud.verification_mode = "existing"
        ud.verification_state = "existing_patient_otp_pending"
        begin_verification_flow(ud)
        payload = {
            "phoneNumber": phone,
            "_auth": ud.active_token,
        }
        if ud.verification_correlation_id:
            payload["_verificationCorrelationId"] = ud.verification_correlation_id
        raw = await _client(context).call_tool("start_existing_phone_verification", payload)
        task = getattr(ud, "otp_collection_task", None)
        if task is None or task.done():
            ud.otp_collection_task = asyncio.create_task(
                wait_for_sms_verification(ud, _client(context), context.session)
            )
        return raw
    except MCPToolError as exc:
        logger.warning("start_existing_phone_verification failed session=%s: %s", ud.session_id, exc)
        return f"I was unable to send a verification code right now. ({exc})"
    except Exception as exc:
        logger.exception("start_existing_phone_verification error session=%s", ud.session_id)
        return "I was unable to send a verification code right now."


@function_tool
async def verify_existing_phone_otp(context: RunContext[SessionUserData], otp: str, phone_number: str = "") -> str:
    """Verify an existing patient by SMS OTP and upgrade the session token."""
    from verification import verify_existing_otp_for_session

    ud = context.userdata
    _ = phone_number
    if not otp.strip():
        return "Please provide the verification code."
    result = await verify_existing_otp_for_session(
        ud,
        _client(context),
        otp,
        channel="spoken",
    )
    return result.message if not result.success else (result.message or "Verified successfully.")


@function_tool
async def collect_verification_code_via_keypad(context: RunContext[SessionUserData]) -> str:
    """Collect a 6-digit verification code using the phone keypad or spoken digits."""
    from livekit.agents.beta.workflows.dtmf_inputs import GetDtmfTask

    from verification import verify_existing_otp_for_session

    ud = context.userdata
    if ud.verification_state != "existing_patient_otp_pending":
        return "Verification is not in progress on this call."
    try:
        result = await GetDtmfTask(
            num_digits=6,
            chat_ctx=context.session.chat_ctx.copy(
                exclude_instructions=True,
                exclude_function_call=True,
                exclude_handoff=True,
                exclude_config_update=True,
            ),
            extra_instructions=(
                "Ask the caller to enter their 6-digit verification code on the keypad, "
                "then press pound. They may also say the digits."
            ),
        )
    except ToolError as exc:
        return exc.message

    verify = await verify_existing_otp_for_session(
        ud,
        _client(context),
        result.user_input,
        channel="get_dtmf_task",
    )
    return verify.message


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


@function_tool
async def list_patient_hospitals(context: RunContext[SessionUserData]) -> str:
    """List Zorba hospitals where the caller has active data-sharing consent.

    Returns patient_id, hospitals (from patient_hospital_consents where revoked_at is NULL),
    and scheduling_requirements (profile email + EMAIL_NOTIFICATION app consent).
    Use this first when scheduling a video visit—not find_nearest_hospital.
    """
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before I can look up your hospitals."
    try:
        return await _client(context).call_tool(
            "list_patient_hospitals",
            {"_auth": ud.active_token},
        )
    except MCPToolError as exc:
        logger.warning("list_patient_hospitals failed session=%s: %s", ud.session_id, exc)
        return str(exc) or "I was unable to look up your hospitals right now."


@function_tool
async def list_schedulable_staff(
    context: RunContext[SessionUserData],
    hospital_id: str,
) -> str:
    """List doctors and nurses available for scheduling at a Zorba hospital.

    Call list_patient_hospitals first, confirm the hospital with the caller, then use
    the hospital_id from that list before schedule_health_staff_meeting.

    Args:
        hospital_id: UUID of the hospital from list_patient_hospitals.
    """
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before I can look up staff."
    try:
        raw = await _client(context).call_tool(
            "list_schedulable_staff",
            {
                "hospitalID": hospital_id,
                "_auth": ud.active_token,
            },
        )
        return raw
    except MCPToolError as exc:
        logger.warning("list_schedulable_staff failed session=%s: %s", ud.session_id, exc)
        return str(exc) or "I was unable to look up hospital staff right now."


@function_tool
async def get_scheduling_clock(context: RunContext[SessionUserData]) -> str:
    """Return the current UTC time when planning a future video visit."""
    now = datetime.now(timezone.utc)
    payload = {
        "utc_now": now.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "hint": (
            "Ask the caller for visit_date (YYYY-MM-DD), visit_time_local (HH:MM 24h), "
            "and their IANA timezone (e.g. America/Chicago). Do not invent RFC3339 strings."
        ),
    }
    return json.dumps(payload)


@function_tool
async def request_staff_transfer(
    context: RunContext[SessionUserData],
    hospital_id: str = "",
    staff_id: str = "",
    transfer_reason: str = "",
    patient_language: str = "",
) -> str:
    """Connect hospital staff into this live phone call for interpreted conversation.

    Use when the verified caller presses 0, asks to speak with a clinician, or needs
    live interpretation with hospital staff on the same call. Prefer the caller's
    detected language when patient_language is empty.

    Args:
        hospital_id: Optional hospital UUID from list_patient_hospitals. Omit when the
            patient has exactly one consented hospital.
        staff_id: Optional preferred staff UUID.
        transfer_reason: Short reason for the transfer.
        patient_language: ISO 639-1 language the patient will speak during interpretation.
    """
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before I can connect hospital staff."
    lang = (patient_language or ud.language or ud.last_user_transcript_language or "en").strip().lower()
    try:
        return await _client(context).call_tool(
            "request_staff_transfer",
            {
                "sessionID": ud.session_id,
                "roomSID": ud.room_sid or ud.session_id,
                "hospitalID": hospital_id.strip(),
                "staffID": staff_id.strip(),
                "transferReason": transfer_reason.strip() or "patient_requested_staff",
                "patientLanguage": lang,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("request_staff_transfer failed session=%s: %s", ud.session_id, exc)
        return str(exc) or "I was unable to connect hospital staff right now."


@function_tool
async def schedule_health_staff_meeting(
    context: RunContext[SessionUserData],
    staff_id: str,
    hospital_id: str,
    visit_date: str,
    visit_time_local: str,
    timezone_name: str,
    patient_confirmed: bool,
    duration_minutes: int = 30,
    title: str = "",
    send_sms: bool = False,
) -> str:
    """Request a video visit with hospital staff after verbal confirmation.

    Only call after list_patient_hospitals and list_schedulable_staff, and after the
    caller confirmed the hospital, staff member, date, and time aloud.

    Args:
        staff_id: UUID of the doctor or nurse from list_schedulable_staff.
        hospital_id: UUID from list_patient_hospitals.
        visit_date: Local calendar date YYYY-MM-DD (confirm with caller).
        visit_time_local: Local start time HH:MM in 24-hour format.
        timezone_name: IANA timezone for the caller (e.g. America/New_York).
        patient_confirmed: Must be true only after the caller verbally confirmed details.
        duration_minutes: Length of the visit in minutes.
        title: Optional visit title.
        send_sms: Whether to send an SMS reminder if the patient consented.
    """
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before I can schedule a visit."
    if not patient_confirmed:
        return (
            "Please confirm the hospital, staff member, date, and time with the caller "
            "before scheduling, then call this tool again with patient_confirmed=true."
        )
    try:
        starts_at = _visit_start_rfc3339(visit_date, visit_time_local, timezone_name)
    except MCPToolError as exc:
        logger.warning("schedule visit time parse failed session=%s: %s", ud.session_id, exc)
        return str(exc)
    try:
        raw = await _client(context).call_tool(
            "schedule_health_staff_meeting",
            {
                "staffID": staff_id,
                "hospitalID": hospital_id,
                "startsAt": starts_at,
                "durationMinutes": duration_minutes,
                "timezone": timezone_name.strip(),
                "title": title,
                "sendSms": send_sms,
                "patientConfirmed": patient_confirmed,
                "_auth": ud.active_token,
            },
        )
        return (
            "Your visit request is pending staff approval. The hospital staff will accept "
            "or reschedule it, and video visit join details will be sent after approval. "
            f"Request details: {raw}"
        )
    except MCPToolError as exc:
        logger.warning("schedule_health_staff_meeting failed session=%s: %s", ud.session_id, exc)
        return str(exc) or "I was unable to schedule that visit right now."


@function_tool
async def list_available_appointment_slots(
    context: RunContext[SessionUserData],
    staff_id: str,
    hospital_id: str,
    from_rfc3339: str = "",
    to_rfc3339: str = "",
) -> str:
    """List bookable appointment slots for staff at a consented hospital."""
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before I can list appointment slots."
    args: dict = {
        "staffID": staff_id.strip(),
        "hospitalID": hospital_id.strip(),
        "_auth": ud.active_token,
    }
    if from_rfc3339.strip():
        args["from"] = from_rfc3339.strip()
    if to_rfc3339.strip():
        args["to"] = to_rfc3339.strip()
    try:
        return await _client(context).call_tool("list_available_appointment_slots", args)
    except MCPToolError as exc:
        logger.warning("list_available_appointment_slots failed session=%s: %s", ud.session_id, exc)
        return str(exc) or "I was unable to list available slots."


@function_tool
async def get_next_available_slot(
    context: RunContext[SessionUserData],
    staff_id: str,
    hospital_id: str,
    after_rfc3339: str = "",
) -> str:
    """Return the earliest bookable appointment slot (auto-select)."""
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before I can find the next slot."
    args: dict = {
        "staffID": staff_id.strip(),
        "hospitalID": hospital_id.strip(),
        "_auth": ud.active_token,
    }
    if after_rfc3339.strip():
        args["after"] = after_rfc3339.strip()
    try:
        return await _client(context).call_tool("get_next_available_slot", args)
    except MCPToolError as exc:
        logger.warning("get_next_available_slot failed session=%s: %s", ud.session_id, exc)
        return str(exc) or "I could not find an available slot."


@function_tool
async def book_appointment(
    context: RunContext[SessionUserData],
    staff_id: str,
    hospital_id: str,
    starts_at: str,
    patient_confirmed: bool,
    duration_minutes: int = 30,
    timezone_name: str = "UTC",
    appointment_type: str = "video",
    title: str = "",
    send_sms: bool = False,
    send_email: bool = True,
) -> str:
    """Book an appointment into an available slot after verbal confirmation."""
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before I can book an appointment."
    if not patient_confirmed:
        return (
            "Please confirm the hospital, staff member, date, and time with the caller "
            "before booking, then call this tool again with patient_confirmed=true."
        )
    try:
        raw = await _client(context).call_tool(
            "book_appointment",
            {
                "staffID": staff_id.strip(),
                "hospitalID": hospital_id.strip(),
                "startsAt": starts_at.strip(),
                "durationMinutes": duration_minutes,
                "timezone": timezone_name.strip() or "UTC",
                "type": appointment_type.strip() or "video",
                "title": title,
                "sendSms": send_sms,
                "sendEmail": send_email,
                "patientConfirmed": patient_confirmed,
                "_auth": ud.active_token,
            },
        )
        return f"Your appointment is booked. Details: {raw}"
    except MCPToolError as exc:
        logger.warning("book_appointment failed session=%s: %s", ud.session_id, exc)
        return str(exc) or "I was unable to book that appointment right now."


@function_tool
async def cancel_appointment(
    context: RunContext[SessionUserData],
    appointment_id: str,
    reason: str = "",
) -> str:
    """Cancel a previously booked appointment for the verified patient."""
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before I can cancel an appointment."
    try:
        return await _client(context).call_tool(
            "cancel_appointment",
            {
                "appointmentID": appointment_id.strip(),
                "reason": reason,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("cancel_appointment failed session=%s: %s", ud.session_id, exc)
        return str(exc) or "I was unable to cancel that appointment."


@function_tool
async def collect_appointment_date_via_keypad(
    context: RunContext[SessionUserData],
    staff_id: str,
    hospital_id: str,
) -> str:
    """Prompt the caller to enter preferred appointment date as MMDD on the keypad, then #."""
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before collecting an appointment date."
    ud.dtmf_mode = "appointment_date"
    ud.dtmf_appointment_date_buffer = ""
    ud.pending_appointment_staff_id = staff_id.strip()
    ud.pending_appointment_hospital_id = hospital_id.strip()
    return (
        "Ask the caller to enter the preferred date on the keypad as four digits MMDD "
        "(month then day), then press #. Press * to clear and start over."
    )


ALL_TOOLS = [
    translate,
    get_location,
    find_nearest_hospital,
    lookup_patient_by_phone,
    start_existing_phone_verification,
    verify_existing_phone_otp,
    collect_verification_code_via_keypad,
    search_health_records,
    answer_health_question,
    log_escalation,
    list_patient_hospitals,
    list_schedulable_staff,
    get_scheduling_clock,
    request_staff_transfer,
    schedule_health_staff_meeting,
    list_available_appointment_slots,
    get_next_available_slot,
    book_appointment,
    cancel_appointment,
    collect_appointment_date_via_keypad,
]
