"""
agent.py - Zorba Health voice agent powered by LiveKit Agents.

SIP dispatch rule dispatches zorba-health-voice → this worker runs the job,
provisions a session token locally, then uses MCP for backend tools.
"""

from __future__ import annotations

import asyncio
import logging
import os
import sys
import textwrap

import aiohttp
from dotenv import load_dotenv
from livekit import rtc
from livekit.agents import (
    Agent,
    AgentServer,
    AgentSession,
    JobContext,
    JobProcess,
    TurnHandlingOptions,
    cli,
    room_io,
)
from livekit.plugins import silero

import config as cfg
import observability
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


class ZorbaAgent(Agent):
    def __init__(self) -> None:
        super().__init__(
            instructions=_ZORBA_INSTRUCTIONS,
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

    await ctx.connect()

    caller_identity = _find_sip_identity(ctx.room)
    if not caller_identity:
        caller_identity = await _wait_for_sip_identity(ctx.room, timeout=15.0)

    room_sid = await _room_sid(ctx.room)

    ud.room_sid = room_sid
    ud.session_id = room_sid
    ud.language = "en"
    ud.provisional_token = auth.mint_provisional_token(room_sid)
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
        try:
            await session.start(
                agent=ZorbaAgent(),
                room=ctx.room,
                room_options=room_io.RoomOptions(
                    audio_input=room_io.AudioInputOptions(),
                ),
            )
            try:
                with tracer.start_as_current_span("voice_agent.greeting"):
                    await session.generate_reply(instructions=_GREETING)
            except Exception:
                logger.exception("greeting failed room=%s", ctx.room.name)
            await session_closed.wait()
        except Exception as exc:
            span.record_exception(exc)
            logger.exception("voice session failed room=%s", ctx.room.name)
            raise
        finally:
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

    from livekit.plugins.turn_detector.multilingual import MultilingualModel

    logger.info("multilingual turn detector enabled")
    return MultilingualModel()


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
