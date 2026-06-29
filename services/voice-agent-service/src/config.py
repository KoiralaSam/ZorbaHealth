"""
config.py – environment variable parsing for the Zorba Health voice agent.

All required variables are validated at import time so the worker fails fast
with a clear message instead of crashing mid-call.
"""

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class WorkerConfig:
    """LiveKit AgentServer / worker pool settings (process start mode)."""

    num_idle_processes: int
    load_threshold: float
    shutdown_process_timeout: float
    initialize_process_timeout: float
    job_memory_warn_mb: float
    job_memory_limit_mb: float


@dataclass(frozen=True)
class Config:
    livekit_url: str
    livekit_api_key: str
    livekit_api_secret: str
    livekit_agent_name: str

    mcp_server_url: str
    patient_service_jwt_secret: str

    # Bridged-call interpretation relay (empty disables segment relaying).
    interpretation_service_url: str
    internal_service_secret: str

    openai_api_key: str
    deepgram_api_key: str
    elevenlabs_api_key: str
    elevenlabs_voice_id: str
    elevenlabs_model: str

    openai_model: str
    tts_provider: str
    deepgram_tts_model: str
    enable_turn_detector: bool
    emergency_transfer_enabled: bool
    emergency_transfer_target: str
    emergency_alert_numbers: tuple[str, ...]
    require_sip_caller: bool
    min_caller_phone_digits: int


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


def _optional_list(name: str) -> tuple[str, ...]:
    raw = os.environ.get(name, "")
    if not raw.strip():
        return ()
    return tuple(part.strip() for part in raw.split(",") if part.strip())


def _optional_float(name: str, default: float) -> float:
    raw = os.environ.get(name, "").strip()
    if not raw:
        return default
    return float(raw)


def _optional_int(name: str, default: int) -> int:
    raw = os.environ.get(name, "").strip()
    if not raw:
        return default
    return int(raw)


def load_worker_config() -> WorkerConfig:
    return WorkerConfig(
        num_idle_processes=_optional_int("VOICE_AGENT_NUM_IDLE_PROCESSES", 1),
        load_threshold=_optional_float("VOICE_AGENT_LOAD_THRESHOLD", 0.75),
        shutdown_process_timeout=_optional_float("VOICE_AGENT_SHUTDOWN_PROCESS_TIMEOUT", 30.0),
        initialize_process_timeout=_optional_float("VOICE_AGENT_INITIALIZE_PROCESS_TIMEOUT", 45.0),
        job_memory_warn_mb=_optional_float("VOICE_AGENT_JOB_MEMORY_WARN_MB", 1200.0),
        job_memory_limit_mb=_optional_float("VOICE_AGENT_JOB_MEMORY_LIMIT_MB", 0.0),
    )


def load() -> Config:
    return Config(
        livekit_url=_require("LIVEKIT_URL"),
        livekit_api_key=_require("LIVEKIT_API_KEY"),
        livekit_api_secret=_require("LIVEKIT_API_SECRET"),
        livekit_agent_name=_optional("LIVEKIT_AGENT_NAME", "zorba-health-voice"),
        mcp_server_url=_require("MCP_SERVER_URL"),
        patient_service_jwt_secret=_require("PATIENT_SERVICE_JWT_SECRET"),
        interpretation_service_url=_optional("INTERPRETATION_SERVICE_URL", ""),
        internal_service_secret=_optional("INTERNAL_SERVICE_SECRET", ""),
        openai_api_key=_require("OPENAI_API_KEY"),
        deepgram_api_key=_require("DEEPGRAM_API_KEY"),
        elevenlabs_api_key=_optional("ELEVENLABS_API_KEY", ""),
        elevenlabs_voice_id=_optional("ELEVENLABS_VOICE_ID", ""),
        elevenlabs_model=_optional("ELEVENLABS_MODEL", "eleven_turbo_v2_5"),
        openai_model=_optional("OPENAI_MODEL", "gpt-4o-mini"),
        tts_provider=_optional("TTS_PROVIDER", "deepgram").lower(),
        deepgram_tts_model=_optional("DEEPGRAM_TTS_MODEL", "aura-2-andromeda-en"),
        enable_turn_detector=_optional_bool("ENABLE_TURN_DETECTOR", False),
        emergency_transfer_enabled=_optional_bool("EMERGENCY_TRANSFER_ENABLED", False),
        emergency_transfer_target=_optional("EMERGENCY_TRANSFER_E164", ""),
        emergency_alert_numbers=_optional_list("EMERGENCY_ALERT_NUMBERS"),
        require_sip_caller=_optional_bool("VOICE_AGENT_REQUIRE_SIP_CALLER", True),
        min_caller_phone_digits=_optional_int("VOICE_AGENT_MIN_CALLER_PHONE_DIGITS", 10),
    )
