"""
bridge_relay.py – producer side of the bridged-call interpretation pipeline.

For every final patient STT utterance, the relay POSTs the segment to the
interpretation service (which loads per-party preferences from Redis and calls
the translation service). Translated segments are published to the LiveKit
room data channel on topic ``zorba.interpretation`` so the hospital staff
client can render live captions.

A bridge session only exists after the patient requests a transfer, so the
relay backs off (``_NO_SESSION_COOLDOWN``) when the interpretation service
reports 404/410 instead of probing on every utterance of every call.
"""

import asyncio
import json
import logging
import time

import aiohttp
from livekit import rtc

logger = logging.getLogger("zorba.bridge_relay")

DATA_TOPIC = "zorba.interpretation"

_NO_SESSION_COOLDOWN = 15.0  # seconds between probes while no bridge exists
_SEGMENT_TIMEOUT = aiohttp.ClientTimeout(total=10.0)


class BridgeRelay:
    """Relays final STT segments to the interpretation service."""

    def __init__(
        self,
        http: aiohttp.ClientSession,
        base_url: str,
        internal_token: str,
        room: rtc.Room,
        session_id: str,
    ) -> None:
        self._http = http
        self._segment_url = base_url.rstrip("/") + "/internal/interpretation/segment"
        self._internal_token = internal_token
        self._room = room
        self._session_id = session_id
        self._suspended_until = 0.0
        self._lock = asyncio.Lock()

    def relay_user_segment(
        self,
        text: str,
        source_lang: str,
        participant: str = "patient",
        target_hint: str = "en",
    ) -> None:
        """Fire-and-forget relay of a final utterance."""
        text = (text or "").strip()
        if not text or time.monotonic() < self._suspended_until:
            return
        asyncio.create_task(
            self.relay_segment(
                text=text,
                source_lang=(source_lang or "").strip().lower(),
                participant=(participant or "").strip().lower() or "patient",
                target_hint=(target_hint or "").strip().lower(),
            )
        )

    async def relay_segment(
        self,
        *,
        text: str,
        source_lang: str,
        participant: str,
        target_hint: str = "",
    ) -> dict | None:
        # Serialize requests so a burst of finals cannot pile up while the
        # service is slow or the session is gone.
        async with self._lock:
            if time.monotonic() < self._suspended_until:
                return None
            headers = {}
            if self._internal_token:
                headers["x-internal-token"] = self._internal_token
            try:
                async with self._http.post(
                    self._segment_url,
                    json={
                        "session_id": self._session_id,
                        "participant": participant,
                        "text": text,
                        "source_lang": source_lang,
                        "target_hint": (target_hint or "").strip().lower(),
                    },
                    headers=headers,
                    timeout=_SEGMENT_TIMEOUT,
                ) as resp:
                    if resp.status in (404, 410):
                        # No active bridge for this room (or it ended).
                        self._suspended_until = time.monotonic() + _NO_SESSION_COOLDOWN
                        return None
                    if resp.status != 200:
                        body = await resp.text()
                        logger.warning(
                            "interpretation relay failed session=%s participant=%s status=%s body=%s",
                            self._session_id,
                            participant,
                            resp.status,
                            body[:200],
                        )
                        return None
                    raw = await resp.text()
                    try:
                        payload = json.loads(raw) if raw.strip() else {}
                    except json.JSONDecodeError:
                        logger.warning(
                            "interpretation relay non-json session=%s participant=%s body=%s",
                            self._session_id,
                            participant,
                            raw[:200],
                        )
                        return None
            except Exception:
                logger.exception(
                    "interpretation relay error session=%s participant=%s",
                    self._session_id,
                    participant,
                )
                return None

        return await self._publish_segment(text, payload, participant)

    async def _publish_segment(
        self,
        original_text: str,
        payload: dict,
        participant: str,
    ) -> dict:
        message = {
            "type": "interpretation.segment",
            "session_id": self._session_id,
            "participant": participant,
            "original_text": original_text,
            "translated_text": payload.get("translated_text", original_text),
            "target_language": payload.get("target_language", ""),
            "passthrough": bool(payload.get("passthrough", False)),
        }
        try:
            await self._room.local_participant.publish_data(
                json.dumps(message).encode("utf-8"),
                reliable=True,
                topic=DATA_TOPIC,
            )
        except Exception:
            logger.exception("interpretation publish failed session=%s", self._session_id)
        return message
