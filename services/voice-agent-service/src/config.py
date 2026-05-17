"""
config.py – environment variable parsing for the Zorba Health voice agent.

All required variables are validated at import time so the worker fails fast
with a clear message instead of crashing mid-call.
"""

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Config:
    livekit_url: str
    livekit_api_key: str
    livekit_api_secret: str
    livekit_agent_name: str

    mcp_server_url: str
    patient_service_jwt_secret: str

    openai_api_key: str
    deepgram_api_key: str
    elevenlabs_api_key: str
    elevenlabs_voice_id: str
    elevenlabs_model: str

    openai_model: str
    tts_provider: str
    deepgram_tts_model: str
    enable_turn_detector: bool


def _require(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        raise RuntimeError(f"Required environment variable {name!r} is not set")
    return value


def _optional(name: str, default: str = "") -> str:
    return os.environ.get(name, default).strip()


def _optional_bool(name: str, default: bool = False) -> bool:
    value = os.environ.get(name)
    if value is None:
        return default
    return value.strip().lower() in {"1", "true", "yes", "on"}


def load() -> Config:
    return Config(
        livekit_url=_require("LIVEKIT_URL"),
        livekit_api_key=_require("LIVEKIT_API_KEY"),
        livekit_api_secret=_require("LIVEKIT_API_SECRET"),
        livekit_agent_name=_optional("LIVEKIT_AGENT_NAME", "zorba-health-voice"),
        mcp_server_url=_require("MCP_SERVER_URL"),
        patient_service_jwt_secret=_require("PATIENT_SERVICE_JWT_SECRET"),
        openai_api_key=_require("OPENAI_API_KEY"),
        deepgram_api_key=_require("DEEPGRAM_API_KEY"),
        elevenlabs_api_key=_optional("ELEVENLABS_API_KEY", ""),
        elevenlabs_voice_id=_optional("ELEVENLABS_VOICE_ID", ""),
        elevenlabs_model=_optional("ELEVENLABS_MODEL", "eleven_turbo_v2_5"),
        openai_model=_optional("OPENAI_MODEL", "gpt-4o"),
        tts_provider=_optional("TTS_PROVIDER", "deepgram").lower(),
        deepgram_tts_model=_optional("DEEPGRAM_TTS_MODEL", "aura-2-andromeda-en"),
        enable_turn_detector=_optional_bool("ENABLE_TURN_DETECTOR", True),
    )
