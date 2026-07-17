"""
agent.py - Zorba Health voice agent powered by LiveKit Agents.

SIP dispatch rule dispatches zorba-health-voice → this worker runs the job,
provisions a session token locally, then uses MCP for backend tools.
"""

from __future__ import annotations

import asyncio
from collections.abc import Mapping
import logging
import os
import sys
import textwrap
import json

import aiohttp
from dotenv import load_dotenv
from livekit import rtc
from livekit.agents import (
    Agent,
    AgentServer,
    AgentSession,
    ConversationItemAddedEvent,
    JobContext,
    JobProcess,
    TurnHandlingOptions,
    UserInputTranscribedEvent,
    cli,
    room_io,
)
from livekit.plugins import silero
from livekit.plugins.turn_detector.multilingual import MultilingualModel

import config as cfg
import observability
import safety
from auth import SessionAuth
from tools.mcp_client import MCPClient
from tools.zorba_tools import ALL_TOOLS
from userdata import SessionUserData
from webhook import maybe_start_webhook_server

load_dotenv(".env.local")

LIVEKIT_AGENT_NAME = os.environ.get("LIVEKIT_AGENT_NAME", "zorba-health-voice")

logger = logging.getLogger("zorba.agent")
tracer = observability.tracer("voice-agent-service.agent")

_ZORBA_INSTRUCTIONS = textwrap.dedent(
    """\
    You are Zorba, a friendly, reliable AI health assistant for Zorba Health.
    You are talking with a patient or caller over the phone.

    # Core policy

    - Start with a short greeting and ask how you can help.
    - You may answer general health questions, provide emergency guidance, and help with
      location or translation without verifying the caller's identity.
    - Before accessing ANY personal records, medications, appointments, test results,
      lab work, or the caller's health profile, you MUST verify their identity.
    - If the caller asks for personal health information and is not yet verified,
      explain that identity verification is required before records can be accessed.
    - Only call health-record tools when a verified patient token is present.
    - Never invent clinical facts or record contents.
    - If a tool fails, say the system cannot access that information right now.
    - For emergencies, encourage calling emergency services (9-1-1).

    # Output rules (voice)

    - Plain text only. No markdown, lists, or code.
    - One to three sentences unless the caller asks for more.
    - One question at a time.
    - Do not reveal tool names or internal details.
    """
)

_GREETING = (
    "Greet the caller as Zorba Health. Introduce yourself briefly, ask how you can help, "
    "and mention you can help with general health questions and emergencies, and can verify "
    "identity if they need access to personal medical records."
)


def _welfare_preamble(ud: SessionUserData) -> str:
    context = ud.welfare_check_context
    if not context:
        return ""
    reason = _display_reason(context.get("reason", "welfare check"))
    detail = (context.get("reason_detail") or "").strip()
    scheduled_at = (context.get("scheduled_at") or "").strip()
    timezone = (context.get("timezone") or "").strip()
    detail_text = f" Patient-provided detail: {detail}" if detail else ""
    scheduled_text = ""
    if scheduled_at:
        scheduled_text = f" The scheduled check time is {scheduled_at}"
        if timezone:
            scheduled_text += f" ({timezone})"
        scheduled_text += "."
    return (
        "This is a patient-scheduled welfare check. The caller is pre-authorized for this scheduled session, "
        "and a verified patient token is already available for record access when clinically relevant. "
        f"The selected reason is {reason}."
        + detail_text
        + scheduled_text
    )


def _safety_preamble(ud: SessionUserData) -> str:
    if not ud.escalation_triggered:
        return ""
    transfer_text = ""
    if ud.transfer_requested and ud.transfer_target:
        transfer_text = (
            f" A transfer to the designated emergency line at {ud.transfer_target} has been requested. "
            "Tell the caller you are connecting urgent help now."
        )
    return (
        "Emergency escalation has already been triggered for this call. "
        "Do not continue casual triage. Give short emergency guidance only, tell the caller to call 9-1-1 immediately, "
        "and keep responses brief and focused on urgent next steps."
        + transfer_text
    )


class ZorbaAgent(Agent):
    def __init__(self, userdata: SessionUserData) -> None:
        super().__init__(
            instructions=_ZORBA_INSTRUCTIONS + "\n\n" + _safety_preamble(userdata),
            tools=ALL_TOOLS,
        )


server = AgentServer()


def prewarm(proc: JobProcess) -> None:
    proc.userdata["vad"] = silero.VAD.load()


server.setup_fnc = prewarm


@server.rtc_session(agent_name=LIVEKIT_AGENT_NAME)
async def zorba_session(ctx: JobContext) -> None:
    observability.configure_tracing()
    ctx.log_context_fields = {"room": ctx.room.name}

    try:
        settings = cfg.load()
    except RuntimeError as exc:
        logger.error("configuration error: %s", exc)
        return
    logger.info("voice-agent-service config loaded tts_provider=%s", settings.tts_provider)

    http_session = aiohttp.ClientSession()
    mcp_client = MCPClient(endpoint=settings.mcp_server_url, session=http_session)
    auth = SessionAuth(settings.patient_service_jwt_secret)

    ud = SessionUserData(room_name=ctx.room.name, mcp_client=mcp_client)
    ud.session_auth = auth
    _apply_welfare_metadata(ctx, ud)

    await ctx.connect()

    caller_identity = _find_sip_identity(ctx.room)
    if not caller_identity:
        caller_identity = await _wait_for_sip_identity(ctx.room, timeout=15.0)

    room_sid = await _room_sid(ctx.room)

    ud.room_sid = room_sid
    ud.session_id = room_sid
    ud.language = _find_sip_language(ctx.room) or "en"
    ud.transfer_requested = settings.emergency_transfer_enabled
    ud.transfer_target = settings.emergency_transfer_target
    ud.alert_phone_numbers = list(settings.emergency_alert_numbers)
    ud.provisional_token = auth.mint_provisional_token(room_sid)
    _finalize_welfare_session_token(ud)
    if caller_identity:
        ud.caller_phone = _extract_phone(caller_identity)
        logger.info(
            "session provisioned room=%s session_id=%s caller_phone=%s",
            ctx.room.name,
            ud.session_id,
            ud.caller_phone,
        )
    else:
        logger.warning("no SIP caller in room=%s", ctx.room.name)

    session = AgentSession[SessionUserData](
        stt=_build_stt(settings),
        llm=_build_llm(settings),
        tts=_build_tts(settings),
        # Self-hosted LiveKit (no Cloud agent-gateway): keep VAD as the default
        # endpointing path. The multilingual turn detector is useful but expensive
        # enough to starve small local Kubernetes nodes during SIP calls.
        turn_handling=TurnHandlingOptions(
            turn_detection=_build_turn_detector(settings),
            interruption={"mode": "vad"},
        ),
        vad=ctx.proc.userdata["vad"],
        userdata=ud,
    )

    session_closed = asyncio.Event()

    @session.on("close")
    def _on_session_close(_ev) -> None:
        session_closed.set()

    async def _handle_user_input_transcribed(ev: UserInputTranscribedEvent) -> None:
        if not ev.is_final:
            return
        transcript = ev.transcript.strip()
        if not transcript:
            return
        ud.last_user_transcript = transcript
        if ev.language:
            ud.last_user_transcript_language = str(ev.language)
            if ud.language in {"", "en"} and str(ev.language) not in {"", "en"}:
                ud.language = str(ev.language)

        decision = safety.evaluate_text(transcript)
        if not decision.should_escalate or ud.escalation_triggered:
            return

        ud.escalation_triggered = True
        ud.escalation_reason = decision.reason
        ud.escalation_guidance = decision.guidance
        try:
            await log_emergency_escalation(ud)
            await session.current_agent.update_instructions(_agent_instructions(ud))
        except Exception:
            logger.exception("failed to handle escalation session=%s", ud.session_id)

    @session.on("user_input_transcribed")
    def _on_user_input_transcribed(ev: UserInputTranscribedEvent) -> None:
        asyncio.create_task(_handle_user_input_transcribed(ev))

    async def _handle_conversation_item_added(ev: ConversationItemAddedEvent) -> None:
        item = ev.item
        role = getattr(item, "role", "")
        if role != "assistant" or ud.escalation_triggered:
            return
        if ud.language not in {"", "en"}:
            try:
                await session.current_agent.update_instructions(_agent_instructions(ud))
            except Exception:
                logger.exception("language instruction refresh failed session=%s", ud.session_id)

    @session.on("conversation_item_added")
    def _on_conversation_item_added(ev: ConversationItemAddedEvent) -> None:
        asyncio.create_task(_handle_conversation_item_added(ev))

    with tracer.start_as_current_span(
        "voice_agent.session",
        attributes={
            "livekit.room.name": ctx.room.name,
            "livekit.room.sid": ud.room_sid,
            "voice.session_id": ud.session_id,
            "voice.caller_phone_present": bool(ud.caller_phone),
            "voice.tts_provider": settings.tts_provider,
        },
    ) as span:
        welfare_terminal_status = "completed"
        welfare_terminal_reason = ""
        try:
            await session.start(
                agent=ZorbaAgent(ud),
                room=ctx.room,
                room_options=room_io.RoomOptions(
                    audio_input=room_io.AudioInputOptions(),
                ),
            )
            try:
                with tracer.start_as_current_span("voice_agent.greeting"):
                    greeting = _GREETING
                    if ud.welfare_check_context:
                        reason = _display_reason(
                            ud.welfare_check_context.get("reason")
                            or ud.welfare_check_context.get("reason_code")
                            or "welfare check"
                        )
                        detail = (ud.welfare_check_context.get("reason_detail") or "").strip()
                        greeting = (
                            "Greet the patient as Zorba Health and say this is their scheduled welfare check "
                            f"for {reason}. "
                        )
                        if detail:
                            greeting += f"Mention their note briefly: {detail}. "
                        greeting += "Ask one open question about how they are doing right now."
                        if caller_identity:
                            await _report_welfare_run_status(ud, mcp_client, "answered")
                        else:
                            welfare_terminal_status = "missed"
                            welfare_terminal_reason = "no_sip_participant"
                    elif ud.escalation_triggered and ud.escalation_reason:
                        greeting = (
                            "Tell the caller this may be a medical emergency because of "
                            f"{ud.escalation_reason}. "
                            + (ud.escalation_guidance or "Advise them to call 9-1-1 immediately.")
                        )
                    elif ud.language not in {"", "en"}:
                        greeting = (
                            f"The caller language is {ud.language}. Greet them in that language, keep the greeting brief, "
                            "and continue helping them in the same language unless they ask to switch."
                        )
                    await session.generate_reply(instructions=greeting)
            except Exception:
                logger.exception("greeting failed room=%s", ctx.room.name)
                if ud.welfare_check_context:
                    welfare_terminal_status = "failed"
                    welfare_terminal_reason = "greeting_failed"
            await session_closed.wait()
        except Exception as exc:
            span.record_exception(exc)
            logger.exception("voice session failed room=%s", ctx.room.name)
            if ud.welfare_check_context:
                welfare_terminal_status = "failed"
                welfare_terminal_reason = str(exc)[:500] or "session_failed"
            raise
        finally:
            if ud.welfare_check_context:
                await _report_welfare_run_status(
                    ud,
                    mcp_client,
                    welfare_terminal_status,
                    welfare_terminal_reason,
                )
            if ud.is_verified:
                from tools.zorba_tools import notify_call_lifecycle_for_session

                try:
                    await notify_call_lifecycle_for_session(ud, mcp_client, "call.ended")
                except Exception:
                    logger.exception("call.ended publish failed session=%s", ud.session_id)
            await mcp_client.close()
            await http_session.close()


async def _room_sid(room: rtc.Room) -> str:
    """Room SID is async in livekit-rtc; never pass the coroutine into JSON."""
    sid = await room.sid
    return sid or room.name


def _find_sip_identity(room: rtc.Room) -> str | None:
    for participant in room.remote_participants.values():
        identity = participant.identity or ""
        if identity.startswith("sip_") or _looks_like_phone(identity):
            return identity
    return None


def _find_sip_language(room: rtc.Room) -> str | None:
    for participant in room.remote_participants.values():
        identity = participant.identity or ""
        if not (identity.startswith("sip_") or _looks_like_phone(identity)):
            continue
        metadata = getattr(participant, "metadata", "") or ""
        if not metadata:
            continue
        try:
            parsed = json.loads(metadata)
        except json.JSONDecodeError:
            continue
        for key in ("language", "preferred_language", "locale"):
            value = str(parsed.get(key) or "").strip().lower()
            if value:
                return value.split("-")[0]
    return None


def _apply_welfare_metadata(ctx: JobContext, ud: SessionUserData) -> None:
    metadata = _load_job_metadata(ctx)
    if not metadata or metadata.get("type") != "welfare_check":
        return

    patient_id = str(metadata.get("patient_id") or "").strip()
    patient_token = str(metadata.get("patient_token") or "").strip()
    request_id = str(metadata.get("request_id") or "").strip()
    run_id = str(metadata.get("run_id") or "").strip()
    if not patient_id or not patient_token or not request_id or not run_id:
        logger.warning("ignoring incomplete welfare-check metadata room=%s", ud.room_name)
        return

    reason = str(metadata.get("reason_code") or metadata.get("reason") or "").strip()
    context = {
        "request_id": request_id,
        "run_id": run_id,
        "patient_id": patient_id,
        "reason": reason,
        "reason_code": reason,
        "reason_detail": str(metadata.get("reason_detail") or "").strip(),
        "scheduled_at": str(metadata.get("scheduled_at") or "").strip(),
        "timezone": str(metadata.get("timezone") or "").strip(),
    }
    # Token stays agent-private; never read SIP participant metadata for auth.
    ud.welfare_check_context = context
    ud.patient_id_hint = patient_id
    ud.upgrade(patient_id, patient_token)
    logger.info(
        "scheduled welfare-check context applied request_id=%s run_id=%s patient_id=%s",
        request_id,
        run_id,
        patient_id,
    )


def _finalize_welfare_session_token(ud: SessionUserData) -> None:
    """Remint a voice-scoped token so MCP session checks match this LiveKit call."""
    if not ud.welfare_check_context or not ud.verified_patient_id or ud.session_auth is None:
        return
    if not ud.session_id:
        return
    voice_token = ud.session_auth.mint_patient_token(
        patient_id=ud.verified_patient_id,
        session_id=ud.session_id,
        scopes=["location:read", "records:read"],
    )
    ud.upgrade(ud.verified_patient_id, voice_token)


async def _report_welfare_run_status(
    ud: SessionUserData,
    mcp_client: MCPClient,
    status: str,
    reason: str = "",
) -> None:
    context = ud.welfare_check_context
    if not context or not ud.verified_patient_id:
        return
    run_id = (context.get("run_id") or "").strip()
    patient_id = (context.get("patient_id") or ud.verified_patient_id or "").strip()
    if not run_id or not patient_id:
        return
    try:
        await mcp_client.call_tool(
            "update_welfare_run_status",
            {
                "patientID": patient_id,
                "runID": run_id,
                "status": status,
                "reason": reason,
                "_auth": ud.active_token,
            },
        )
    except Exception:
        logger.exception(
            "welfare run status update failed status=%s run_id=%s session=%s",
            status,
            run_id,
            ud.session_id,
        )


def _load_job_metadata(ctx: JobContext) -> dict[str, str]:
    raw = getattr(getattr(ctx, "job", None), "metadata", "") or ""
    if not raw:
        return {}
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        logger.warning("invalid LiveKit job metadata json room=%s", getattr(ctx.room, "name", ""))
        return {}
    if not isinstance(parsed, Mapping):
        return {}
    return {str(key): str(value) for key, value in parsed.items() if value is not None}


async def _wait_for_sip_identity(room: rtc.Room, timeout: float = 15.0) -> str | None:
    deadline = asyncio.get_event_loop().time() + timeout
    while asyncio.get_event_loop().time() < deadline:
        identity = _find_sip_identity(room)
        if identity:
            return identity
        await asyncio.sleep(0.5)
    return None


def _looks_like_phone(identity: str) -> bool:
    digits = "".join(c for c in identity if c.isdigit())
    return len(digits) >= 10


def _extract_phone(identity: str) -> str:
    return "".join(c for c in identity.removeprefix("sip_") if c.isdigit())


def _build_stt(settings: cfg.Config):
    from livekit.plugins import deepgram

    return deepgram.STT(
        model="nova-2",
        language="multi",
        api_key=settings.deepgram_api_key,
    )


def _build_llm(settings: cfg.Config):
    from livekit.plugins import openai

    return openai.LLM(
        model=settings.openai_model,
        api_key=settings.openai_api_key,
    )


def _agent_instructions(ud: SessionUserData) -> str:
    parts = [_ZORBA_INSTRUCTIONS]
    welfare = _welfare_preamble(ud)
    if welfare:
        parts.append(welfare)
    if ud.language not in {"", "en"}:
        parts.append(
            f"The caller's preferred language is {ud.language}. Speak in {ud.language} unless they ask to switch languages."
        )
    preamble = _safety_preamble(ud)
    if preamble:
        parts.append(preamble)
    return "\n\n".join(parts)


def _display_reason(reason: str) -> str:
    return reason.replace("_", " ").strip() or "welfare check"


async def log_emergency_escalation(ud: SessionUserData) -> None:
    client = ud.mcp_client
    if client is None:
        raise RuntimeError("MCP client is not configured")
    await client.call_tool(
        "log_escalation",
        {
            "sessionID": ud.session_id,
            "patientID": ud.verified_patient_id or "",
            "callerPhone": ud.caller_phone,
            "reason": ud.escalation_reason,
            "severity": "high",
            "transferRequested": ud.transfer_requested,
            "transferTarget": ud.transfer_target,
            "alertPhoneNumbers": ud.alert_phone_numbers or [],
            "transcriptExcerpt": ud.last_user_transcript,
            "_auth": ud.active_token,
        },
    )


def _build_tts(settings: cfg.Config):
    if settings.tts_provider == "deepgram":
        from livekit.plugins import deepgram

        logger.info("using Deepgram TTS model=%s", settings.deepgram_tts_model)
        return deepgram.TTS(
            api_key=settings.deepgram_api_key,
            model=settings.deepgram_tts_model,
        )

    if settings.tts_provider != "elevenlabs":
        raise RuntimeError(f"unsupported TTS_PROVIDER {settings.tts_provider!r}")

    if not settings.elevenlabs_api_key:
        raise RuntimeError("ELEVENLABS_API_KEY is required when TTS_PROVIDER=elevenlabs")

    from livekit.plugins import elevenlabs

    kwargs = {
        "api_key": settings.elevenlabs_api_key,
        "model": settings.elevenlabs_model,
    }
    if _configured_value(settings.elevenlabs_voice_id):
        kwargs["voice_id"] = settings.elevenlabs_voice_id
    logger.info(
        "using ElevenLabs TTS model=%s voice_configured=%s",
        settings.elevenlabs_model,
        "voice_id" in kwargs,
    )
    return elevenlabs.TTS(**kwargs)


def _build_turn_detector(settings: cfg.Config):
    if not settings.enable_turn_detector:
        logger.info("multilingual turn detector disabled; using VAD endpointing")
        return None

    logger.info("multilingual turn detector enabled")
    try:
        return MultilingualModel()
    except RuntimeError:
        logger.exception("multilingual turn detector unavailable; falling back to VAD endpointing")
        return None


def _configured_value(value: str) -> bool:
    value = value.strip()
    return bool(value) and not value.upper().startswith("REPLACE_")


if __name__ == "__main__":
    observability.configure_tracing()
    webhook_server = None
    if "download-files" not in sys.argv[1:]:
        webhook_server = maybe_start_webhook_server()
    try:
        cli.run_app(server)
    finally:
        if webhook_server is not None:
            webhook_server.close()
        observability.shutdown_tracing()
