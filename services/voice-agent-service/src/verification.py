from __future__ import annotations

import asyncio
import json
import logging
import re
import uuid
from dataclasses import dataclass
from typing import TYPE_CHECKING, Any

import observability
from tools.mcp_client import MCPToolError

if TYPE_CHECKING:
    from livekit.agents import AgentSession
    from tools.mcp_client import MCPClient
    from userdata import SessionUserData

logger = logging.getLogger("zorba.verification")
tracer = observability.tracer("voice-agent-service.verification")

OTP_LENGTH = 6
_RE_OTP = re.compile(r"^\d{6}$")


@dataclass
class VerifyResult:
    success: bool
    message: str = ""
    patient_id: str = ""


def normalize_otp(value: str) -> str | None:
    digits = "".join(c for c in value if c.isdigit())
    if len(digits) == OTP_LENGTH and _RE_OTP.match(digits):
        return digits
    return None


def begin_verification_flow(ud: SessionUserData) -> None:
    if not ud.verification_correlation_id:
        ud.verification_correlation_id = str(uuid.uuid4())


def _audit_payload(ud: SessionUserData, channel: str) -> dict[str, Any]:
    payload: dict[str, Any] = {"_auth": ud.active_token}
    if ud.verification_correlation_id:
        payload["_verificationCorrelationId"] = ud.verification_correlation_id
    if channel:
        payload["verificationChannel"] = channel
    return payload


async def verify_existing_otp_for_session(
    ud: SessionUserData,
    mcp_client: MCPClient,
    otp: str,
    *,
    channel: str,
) -> VerifyResult:
    from tools.zorba_tools import _apply_verified_session, _voice_caller_phone, notify_call_lifecycle_for_session

    normalized = normalize_otp(otp)
    with tracer.start_as_current_span(
        "voice.verification.attempt",
        attributes={
            "voice.session_id": ud.session_id,
            "voice.verification.channel": channel,
            "voice.caller_phone_present": bool(ud.caller_phone),
        },
    ) as span:
        if not normalized:
            span.set_attribute("voice.verification.outcome", "invalid_format")
            return VerifyResult(success=False, message="Please provide a 6-digit verification code.")

        phone = _voice_caller_phone(ud)
        if not phone:
            span.set_attribute("voice.verification.outcome", "no_caller_phone")
            return VerifyResult(success=False, message="I need a phone number before I can verify that code.")

        payload = {
            "phoneNumber": phone,
            "otp": normalized,
            **_audit_payload(ud, channel),
        }
        try:
            raw = await mcp_client.call_tool("verify_existing_phone_otp", payload)
        except MCPToolError as exc:
            span.set_attribute("voice.verification.outcome", "mcp_error")
            logger.warning("verify_existing_phone_otp failed session=%s channel=%s: %s", ud.session_id, channel, exc)
            ud.verification_attempts += 1
            return VerifyResult(success=False, message="That verification code did not work.")

        try:
            body = json.loads(raw)
        except json.JSONDecodeError:
            span.set_attribute("voice.verification.outcome", "mcp_error")
            return VerifyResult(success=False, message=raw or "Verification failed.")

        patient_id = str(body.get("patientID") or "").strip()
        access_token = str(body.get("accessToken") or "").strip()
        if patient_id and access_token:
            _apply_verified_session(ud, patient_id, access_token)
            await notify_call_lifecycle_for_session(ud, mcp_client, "call.started")
            ud.dtmf_otp_buffer = ""
            ud.verification_state = "verified_patient"
            span.set_attribute("voice.verification.outcome", "success")
            return VerifyResult(
                success=True,
                message=str(body.get("message") or "Verified successfully."),
                patient_id=patient_id,
            )

        span.set_attribute("voice.verification.outcome", "mcp_error")
        ud.verification_attempts += 1
        return VerifyResult(success=False, message=str(body.get("message") or "That verification code did not work."))


async def consume_sms_verification_once(ud: SessionUserData, mcp_client: MCPClient) -> VerifyResult:
    from tools.zorba_tools import _apply_verified_session, notify_call_lifecycle_for_session

    with tracer.start_as_current_span(
        "voice.verification.attempt",
        attributes={
            "voice.session_id": ud.session_id,
            "voice.verification.channel": "sms_poll",
            "voice.caller_phone_present": bool(ud.caller_phone),
        },
    ) as span:
        try:
            raw = await mcp_client.call_tool(
                "consume_voice_verification",
                _audit_payload(ud, "sms_poll"),
            )
        except MCPToolError:
            span.set_attribute("voice.verification.outcome", "mcp_error")
            return VerifyResult(success=False)

        try:
            body = json.loads(raw)
        except json.JSONDecodeError:
            span.set_attribute("voice.verification.outcome", "pending")
            return VerifyResult(success=False)

        if not body.get("verified"):
            span.set_attribute("voice.verification.outcome", "pending")
            return VerifyResult(success=False)

        patient_id = str(body.get("patientID") or "").strip()
        if not patient_id or ud.session_auth is None:
            span.set_attribute("voice.verification.outcome", "mcp_error")
            return VerifyResult(success=False, message="Verification incomplete.")

        voice_token = ud.session_auth.mint_patient_token(
            patient_id=patient_id,
            session_id=ud.session_id,
            scopes=["location:read", "records:read"],
        )
        _apply_verified_session(ud, patient_id, voice_token)
        await notify_call_lifecycle_for_session(ud, mcp_client, "call.started")
        ud.dtmf_otp_buffer = ""
        span.set_attribute("voice.verification.outcome", "success")
        return VerifyResult(success=True, message="Verified via text message.", patient_id=patient_id)


async def wait_for_sms_verification(
    ud: SessionUserData,
    mcp_client: MCPClient,
    session: AgentSession,
    *,
    timeout: float = 300.0,
    interval: float = 2.0,
) -> None:
    deadline = asyncio.get_event_loop().time() + timeout
    while asyncio.get_event_loop().time() < deadline:
        if ud.is_verified:
            return
        if ud.verification_state != "existing_patient_otp_pending":
            return
        result = await consume_sms_verification_once(ud, mcp_client)
        if result.success:
            try:
                await session.generate_reply(
                    instructions="Tell the caller briefly that their identity was verified successfully.",
                    allow_interruptions=False,
                )
            except Exception:
                logger.exception("sms verification confirmation failed session=%s", ud.session_id)
            return
        await asyncio.sleep(interval)
