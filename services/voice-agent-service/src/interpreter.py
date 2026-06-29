"""
interpreter.py - bidirectional bridge interpreter runtime for doctor consults.

This controller suppresses the assistant while a staff participant is present,
relays patient/staff speech through interpretation-service, and speaks the
doctor->patient direction back into the LiveKit room with TTS.
"""

from __future__ import annotations

import asyncio
import logging
import time
from dataclasses import dataclass

import aiohttp
import jwt
from livekit import rtc
from livekit.agents import stt
from livekit.plugins import deepgram

import config as cfg
from bridge_relay import BridgeRelay
from userdata import SessionUserData

logger = logging.getLogger("zorba.interpreter")

_STAFF_PREFIX = "staff-"
_TWIRP_SUBSCRIPTIONS_PATH = "/twirp/livekit.RoomService/UpdateSubscriptions"


def is_staff_identity(identity: str) -> bool:
    return (identity or "").strip().lower().startswith(_STAFF_PREFIX)


def _http_base_url(value: str) -> str:
    value = (value or "").strip()
    if value.startswith("ws://"):
        return "http://" + value.removeprefix("ws://")
    if value.startswith("wss://"):
        return "https://" + value.removeprefix("wss://")
    return value.rstrip("/")


def _language_hint(value: str, fallback: str = "en") -> str:
    value = (value or "").strip()
    return value or fallback


@dataclass
class _TrackRouting:
    participant_identity: str
    track_sid: str


class LiveKitRoomAdmin:
    """Small Twirp client for the single subscription API we need."""

    def __init__(
        self,
        *,
        http: aiohttp.ClientSession,
        livekit_url: str,
        api_key: str,
        api_secret: str,
    ) -> None:
        self._http = http
        self._base_url = _http_base_url(livekit_url)
        self._api_key = api_key
        self._api_secret = api_secret

    @property
    def enabled(self) -> bool:
        return bool(self._base_url and self._api_key and self._api_secret)

    def _headers(self, room_name: str) -> dict[str, str]:
        now = int(time.time())
        token = jwt.encode(
            {
                "iss": self._api_key,
                "sub": "zorba-health-voice-admin",
                "nbf": now - 5,
                "exp": now + 300,
                "video": {
                    "room": room_name,
                    "roomAdmin": True,
                },
            },
            self._api_secret,
            algorithm="HS256",
        )
        return {
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        }

    async def update_subscriptions(
        self,
        *,
        room_name: str,
        participant_identity: str,
        track_sids: list[str],
        subscribe: bool,
    ) -> None:
        if not self.enabled or not participant_identity or not track_sids:
            return
        url = self._base_url.rstrip("/") + _TWIRP_SUBSCRIPTIONS_PATH
        async with self._http.post(
            url,
            json={
                "room": room_name,
                "identity": participant_identity,
                "trackSids": track_sids,
                "subscribe": subscribe,
            },
            headers=self._headers(room_name),
            timeout=aiohttp.ClientTimeout(total=10.0),
        ) as resp:
            if resp.status == 200:
                return
            body = await resp.text()
            raise RuntimeError(
                f"update_subscriptions status={resp.status} body={body[:200]}"
            )


class InterpreterController:
    """Owns bridged-call interpreter mode while staff are present."""

    def __init__(
        self,
        *,
        session,
        room: rtc.Room,
        userdata: SessionUserData,
        settings: cfg.Config,
        bridge_relay: BridgeRelay,
        http_session: aiohttp.ClientSession,
    ) -> None:
        self._session = session
        self._room = room
        self._ud = userdata
        self._settings = settings
        self._bridge_relay = bridge_relay
        self._room_admin = LiveKitRoomAdmin(
            http=http_session,
            livekit_url=settings.livekit_url,
            api_key=settings.livekit_api_key,
            api_secret=settings.livekit_api_secret,
        )

        self._lock = asyncio.Lock()
        self._tts_lock = asyncio.Lock()
        self._closing = False
        self._staff_identity = ""
        self._staff_task: asyncio.Task | None = None
        self._patient_task: asyncio.Task | None = None
        self._route_state: dict[str, _TrackRouting] = {}

    @property
    def active(self) -> bool:
        return self._ud.interpreter_mode

    async def aclose(self) -> None:
        self._closing = True
        await self.exit_mode()

    async def enter_mode(self, participant: rtc.RemoteParticipant) -> None:
        identity = (participant.identity or "").strip()
        if not is_staff_identity(identity):
            return

        async with self._lock:
            if self._closing:
                return
            if self._ud.interpreter_mode and self._staff_identity == identity:
                await self._route_staff_audio(participant)
                return

            await self._stop_tasks()
            self._staff_identity = identity
            self._ud.interpreter_mode = True
            self._ud.staff_identity = identity
            await self._suppress_assistant(True)

            self._staff_task = asyncio.create_task(
                self._run_staff_stream(participant),
                name="bridge_staff_stream",
            )
            patient = self._patient_participant()
            if patient is not None:
                self._patient_task = asyncio.create_task(
                    self._run_patient_stream(patient),
                    name="bridge_patient_stream",
                )
            await self._route_staff_audio(participant)
            logger.info(
                "interpreter mode enabled session=%s staff=%s",
                self._ud.session_id,
                identity,
            )

    async def exit_mode(self) -> None:
        async with self._lock:
            if not self._ud.interpreter_mode and not self._staff_identity:
                return
            await self._stop_tasks()
            await self._restore_routes()
            await self._suppress_assistant(False)
            logger.info(
                "interpreter mode disabled session=%s staff=%s",
                self._ud.session_id,
                self._staff_identity,
            )
            self._staff_identity = ""
            self._ud.interpreter_mode = False
            self._ud.staff_identity = ""

    async def on_track_published(
        self,
        publication: rtc.RemoteTrackPublication,
        participant: rtc.RemoteParticipant,
    ) -> None:
        if not self._ud.interpreter_mode:
            return
        if participant.identity != self._staff_identity:
            return
        if publication.kind != rtc.TrackKind.KIND_AUDIO:
            return
        await self._route_staff_audio(participant)

    def _patient_participant(self) -> rtc.RemoteParticipant | None:
        identity = (self._ud.caller_identity or "").strip()
        if not identity:
            return None
        return self._room.remote_participants.get(identity)

    async def _stop_tasks(self) -> None:
        for task in (self._staff_task, self._patient_task):
            if task is None:
                continue
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass
            except Exception:
                logger.exception("interpreter stream task failed session=%s", self._ud.session_id)
        self._staff_task = None
        self._patient_task = None

    async def _suppress_assistant(self, active: bool) -> None:
        try:
            await self._session.interrupt(force=True)
        except Exception:
            logger.debug("session interrupt failed during interpreter toggle", exc_info=True)

        activity = getattr(self._session, "_activity", None)
        if activity is not None and hasattr(activity, "_new_turns_blocked"):
            activity._new_turns_blocked = active

    async def _restore_routes(self) -> None:
        if not self._route_state:
            return
        if not self._room_admin.enabled:
            self._route_state.clear()
            return
        track_sids = [item.track_sid for item in self._route_state.values() if item.track_sid]
        if track_sids:
            try:
                await self._room_admin.update_subscriptions(
                    room_name=self._room.name,
                    participant_identity=self._ud.caller_identity,
                    track_sids=track_sids,
                    subscribe=True,
                )
            except Exception:
                logger.exception("failed to restore raw staff subscriptions session=%s", self._ud.session_id)
        self._route_state.clear()

    async def _route_staff_audio(self, participant: rtc.RemoteParticipant) -> None:
        if not self._room_admin.enabled or not self._ud.caller_identity:
            return
        track_sids: list[str] = []
        for publication in participant.track_publications.values():
            if publication.kind != rtc.TrackKind.KIND_AUDIO:
                continue
            if not publication.sid:
                continue
            track_sids.append(publication.sid)
            self._route_state[publication.sid] = _TrackRouting(
                participant_identity=participant.identity,
                track_sid=publication.sid,
            )
        if not track_sids:
            return
        try:
            await self._room_admin.update_subscriptions(
                room_name=self._room.name,
                participant_identity=self._ud.caller_identity,
                track_sids=track_sids,
                subscribe=False,
            )
        except Exception:
            logger.exception("failed to mute raw staff audio for patient session=%s", self._ud.session_id)

    async def _run_patient_stream(self, participant: rtc.RemoteParticipant) -> None:
        await self._run_stream(
            participant=participant,
            participant_role="patient",
            language_hint=_language_hint(self._ud.language),
        )

    async def _run_staff_stream(self, participant: rtc.RemoteParticipant) -> None:
        await self._run_stream(
            participant=participant,
            participant_role="staff",
            language_hint="en",
        )

    async def _run_stream(
        self,
        *,
        participant: rtc.RemoteParticipant,
        participant_role: str,
        language_hint: str,
    ) -> None:
        audio_stream = rtc.AudioStream.from_participant(
            participant=participant,
            track_source=rtc.TrackSource.SOURCE_MICROPHONE,
            sample_rate=16000,
            num_channels=1,
        )
        speech_stream = deepgram.STT(
            model="nova-2",
            language=language_hint,
            api_key=self._settings.deepgram_api_key,
        ).stream(language=language_hint)

        async def _pump_audio() -> None:
            try:
                async for event in audio_stream:
                    speech_stream.push_frame(event.frame)
            finally:
                speech_stream.end_input()
                await audio_stream.aclose()

        async def _consume_transcripts() -> None:
            async for event in speech_stream:
                if event.type != stt.SpeechEventType.FINAL_TRANSCRIPT or not event.alternatives:
                    continue
                alternative = event.alternatives[0]
                transcript = (alternative.text or "").strip()
                if not transcript:
                    continue
                source_lang = str(alternative.language or "").strip().lower()
                await self._handle_final_transcript(
                    participant_role=participant_role,
                    transcript=transcript,
                    source_lang=source_lang,
                )

        pump_task = asyncio.create_task(_pump_audio(), name=f"{participant_role}_audio_pump")
        try:
            await _consume_transcripts()
        finally:
            pump_task.cancel()
            try:
                await pump_task
            except asyncio.CancelledError:
                pass
            await speech_stream.aclose()

    async def _handle_final_transcript(
        self,
        *,
        participant_role: str,
        transcript: str,
        source_lang: str,
    ) -> None:
        if participant_role == "patient":
            self._ud.last_user_transcript = transcript
            if source_lang:
                self._ud.last_user_transcript_language = source_lang
                if self._ud.language in {"", "en"} and source_lang not in {"", "en"}:
                    self._ud.language = source_lang
            await self._bridge_relay.relay_segment(
                text=transcript,
                source_lang=source_lang or self._ud.language,
                participant="patient",
                target_hint="en",
            )
            return

        payload = await self._bridge_relay.relay_segment(
            text=transcript,
            source_lang=source_lang,
            participant="staff",
            target_hint=_language_hint(self._ud.language),
        )
        if not payload:
            return
        translated_text = (payload.get("translated_text") or "").strip()
        if not translated_text:
            return
        async with self._tts_lock:
            await self._session.say(
                translated_text,
                allow_interruptions=False,
                add_to_chat_ctx=False,
            )
