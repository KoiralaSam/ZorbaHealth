from __future__ import annotations

import base64
import hashlib
import json
import logging
import os
import threading
import time
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any

import jwt

logger = logging.getLogger("zorba.webhook")

_MAX_WEBHOOK_BYTES = 1 << 20


def parse_bind_addr(addr: str) -> tuple[str, int]:
    addr = addr.strip()
    if not addr:
        return "0.0.0.0", 8090
    if addr.startswith(":"):
        return "0.0.0.0", int(addr[1:])

    host, sep, port = addr.rpartition(":")
    if not sep:
        raise ValueError(f"invalid bind address {addr!r}")
    host = host or "0.0.0.0"
    return host, int(port)


def verify_webhook_event(
    *,
    api_key: str,
    api_secret: str,
    auth_token: str,
    body: bytes,
) -> dict[str, Any]:
    if not auth_token:
        raise ValueError("missing authorization header")

    unverified = jwt.decode(
        auth_token,
        options={"verify_signature": False, "verify_exp": False, "verify_aud": False},
        algorithms=["HS256"],
    )
    issuer = str(unverified.get("iss") or "")
    if issuer != api_key:
        raise ValueError("unexpected webhook issuer")

    claims = jwt.decode(
        auth_token,
        api_secret,
        algorithms=["HS256"],
        issuer=api_key,
        options={"verify_aud": False},
    )

    checksum = base64.b64encode(hashlib.sha256(body).digest()).decode("ascii")
    if claims.get("sha256") != checksum:
        raise ValueError("invalid webhook checksum")

    payload = json.loads(body.decode("utf-8"))
    if not isinstance(payload, dict):
        raise ValueError("webhook payload must be an object")
    return payload


def is_agent_identity(identity: str) -> bool:
    return identity.startswith("agent-worker-") or identity.startswith("agent-") or identity.startswith(
        "zorba-"
    )


def extract_caller_phone(identity: str) -> str:
    from phone import canonical_phone_digits

    identity = identity.strip()
    if identity.startswith("sip_"):
        identity = identity[len("sip_") :]
    return canonical_phone_digits(identity)


def participant_language(metadata: str) -> str:
    if not metadata:
        return "en"
    try:
        parsed = json.loads(metadata)
    except json.JSONDecodeError:
        return "en"
    language = str(parsed.get("language") or parsed.get("preferred_language") or parsed.get("locale") or "").strip()
    if "-" in language:
        language = language.split("-", 1)[0]
    return language or "en"


def process_webhook_event(event: dict[str, Any]) -> int:
    room = event.get("room") or {}
    participant = event.get("participant") or {}
    room_name = str(room.get("name") or "")
    participant_identity = str(participant.get("identity") or "")

    logger.info(
        "livekit webhook event=%r room=%r participant=%r",
        event.get("event", ""),
        room_name,
        participant_identity,
    )

    if not participant:
        return HTTPStatus.OK

    if is_agent_identity(participant_identity):
        return HTTPStatus.OK

    event_name = str(event.get("event") or "")
    if event_name == "participant_joined":
        caller_phone = extract_caller_phone(participant_identity)
        if len(caller_phone) < 10:
            logger.info(
                "livekit ignoring participant_joined: short/non-phone identity=%r digits=%r",
                participant_identity,
                caller_phone,
            )
            return HTTPStatus.OK

        logger.info(
            "livekit participant joined room=%s room_sid=%s caller_phone=%s language=%s",
            room_name,
            room.get("sid", ""),
            caller_phone,
            participant_language(str(participant.get("metadata") or "")),
        )
        return HTTPStatus.ACCEPTED

    if event_name == "participant_left":
        logger.info("livekit participant left room=%s identity=%s", room_name, participant_identity)
        return HTTPStatus.OK

    return HTTPStatus.OK


class _WebhookRequestHandler(BaseHTTPRequestHandler):
    server: "_WebhookHTTPServer"

    def do_GET(self) -> None:  # noqa: N802
        if self.path != "/health":
            self.send_error(HTTPStatus.NOT_FOUND, "not found")
            return
        self.send_response(HTTPStatus.OK)
        self.end_headers()

    def do_POST(self) -> None:  # noqa: N802
        if self.path != "/webhook/livekit":
            self.send_error(HTTPStatus.NOT_FOUND, "not found")
            return

        length = int(self.headers.get("Content-Length") or "0")
        if length > _MAX_WEBHOOK_BYTES:
            self.send_error(HTTPStatus.REQUEST_ENTITY_TOO_LARGE, "payload too large")
            return

        body = self.rfile.read(length)
        try:
            event = verify_webhook_event(
                api_key=self.server.api_key,
                api_secret=self.server.api_secret,
                auth_token=self.headers.get("Authorization", ""),
                body=body,
            )
        except Exception:
            logger.exception("livekit webhook verification failed")
            self.send_error(HTTPStatus.UNAUTHORIZED, "invalid signature")
            return

        status = process_webhook_event(event)
        self.send_response(status)
        self.end_headers()

    def log_message(self, format: str, *args: Any) -> None:  # noqa: A003
        logger.debug("webhook http: " + format, *args)


class _WebhookHTTPServer(ThreadingHTTPServer):
    def __init__(self, server_address: tuple[str, int], api_key: str, api_secret: str) -> None:
        super().__init__(server_address, _WebhookRequestHandler)
        self.api_key = api_key
        self.api_secret = api_secret


class WebhookServer:
    def __init__(self, http_addr: str, api_key: str, api_secret: str) -> None:
        host, port = parse_bind_addr(http_addr)
        self._server = _WebhookHTTPServer((host, port), api_key=api_key, api_secret=api_secret)
        self._thread = threading.Thread(target=self._server.serve_forever, daemon=True)
        self._bind_addr = f"{host}:{port}"

    def start(self) -> None:
        self._thread.start()
        logger.info("voice-agent-service webhook server listening on %s", self._bind_addr)

    def close(self) -> None:
        self._server.shutdown()
        self._server.server_close()
        self._thread.join(timeout=5)


def maybe_start_webhook_server() -> WebhookServer | None:
    api_key = os.environ.get("LIVEKIT_API_KEY", "").strip()
    api_secret = os.environ.get("LIVEKIT_API_SECRET", "").strip()
    if not api_key or not api_secret:
        logger.warning("voice-agent webhook disabled: LIVEKIT_API_KEY or LIVEKIT_API_SECRET missing")
        return None

    http_addr = os.environ.get("VOICE_AGENT_HTTP_ADDR", ":8090")
    server = WebhookServer(http_addr=http_addr, api_key=api_key, api_secret=api_secret)
    server.start()
    # Give the background server a moment to bind so startup errors surface early.
    time.sleep(0.05)
    return server
