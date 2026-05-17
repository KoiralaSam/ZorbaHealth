import base64
import hashlib
import json
import time
from http import HTTPStatus

import jwt

from webhook import (
    extract_caller_phone,
    is_agent_identity,
    parse_bind_addr,
    process_webhook_event,
    verify_webhook_event,
)


def _signed_token(api_key: str, api_secret: str, body: bytes) -> str:
    sha = base64.b64encode(hashlib.sha256(body).digest()).decode("ascii")
    now = int(time.time())
    return jwt.encode(
        {
            "iss": api_key,
            "exp": now + 300,
            "nbf": now - 5,
            "sha256": sha,
        },
        api_secret,
        algorithm="HS256",
    )


def test_parse_bind_addr_supports_colon_only_port() -> None:
    assert parse_bind_addr(":8090") == ("0.0.0.0", 8090)


def test_verify_webhook_event_accepts_valid_signature() -> None:
    body = json.dumps({"event": "participant_joined"}).encode("utf-8")
    token = _signed_token("key", "secret", body)

    event = verify_webhook_event(
        api_key="key",
        api_secret="secret",
        auth_token=token,
        body=body,
    )

    assert event["event"] == "participant_joined"


def test_process_webhook_event_accepts_real_caller_join() -> None:
    status = process_webhook_event(
        {
            "event": "participant_joined",
            "room": {"name": "zorba-room", "sid": "RM_123"},
            "participant": {"identity": "sip_+13185551212", "metadata": '{"language":"en"}'},
        }
    )

    assert status == HTTPStatus.ACCEPTED


def test_process_webhook_event_ignores_agent_identity() -> None:
    status = process_webhook_event(
        {
            "event": "participant_joined",
            "room": {"name": "zorba-room"},
            "participant": {"identity": "zorba-health-voice"},
        }
    )

    assert status == HTTPStatus.OK
    assert is_agent_identity("zorba-health-voice")
    assert extract_caller_phone("sip_+13185551212") == "13185551212"
