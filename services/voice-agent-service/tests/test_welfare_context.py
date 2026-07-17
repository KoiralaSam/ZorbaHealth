import json
from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest

from agent import (
    _agent_instructions,
    _apply_welfare_metadata,
    _finalize_welfare_session_token,
    _report_welfare_run_status,
    _welfare_preamble,
)
from userdata import SessionUserData


def test_applies_signed_welfare_metadata_as_verified_session() -> None:
    metadata = {
        "type": "welfare_check",
        "request_id": "request-1",
        "run_id": "run-1",
        "patient_id": "patient-1",
        "reason_code": "mental_wellbeing",
        "reason_detail": "Patient asked for a mood check.",
        "scheduled_at": "2026-06-29T14:00:00Z",
        "timezone": "America/Chicago",
        "patient_token": "patient-token",
    }
    ctx = SimpleNamespace(
        job=SimpleNamespace(metadata=json.dumps(metadata)),
        room=SimpleNamespace(name="welfare-room"),
    )
    userdata = SessionUserData(room_name="welfare-room")

    _apply_welfare_metadata(ctx, userdata)

    assert userdata.is_verified is True
    assert userdata.verified_patient_id == "patient-1"
    assert userdata.patient_token == "patient-token"
    assert userdata.patient_id_hint == "patient-1"
    assert userdata.welfare_check_context is not None
    assert userdata.welfare_check_context["reason"] == "mental_wellbeing"
    assert userdata.welfare_check_context["reason_code"] == "mental_wellbeing"
    assert "patient-scheduled welfare check" in _welfare_preamble(userdata)
    assert "mental wellbeing" in _agent_instructions(userdata)


def test_ignores_incomplete_welfare_metadata() -> None:
    ctx = SimpleNamespace(
        job=SimpleNamespace(metadata='{"type":"welfare_check","patient_id":"patient-1"}'),
        room=SimpleNamespace(name="welfare-room"),
    )
    userdata = SessionUserData(room_name="welfare-room")

    _apply_welfare_metadata(ctx, userdata)

    assert userdata.is_verified is False
    assert userdata.welfare_check_context is None


def test_ignores_non_welfare_job_metadata() -> None:
    ctx = SimpleNamespace(
        job=SimpleNamespace(metadata='{"type":"other","patient_token":"x","patient_id":"p","request_id":"r","run_id":"u"}'),
        room=SimpleNamespace(name="room"),
    )
    userdata = SessionUserData(room_name="room")
    _apply_welfare_metadata(ctx, userdata)
    assert userdata.welfare_check_context is None


def test_finalize_welfare_session_token_remints_voice_scoped_token() -> None:
    class FakeAuth:
        def mint_patient_token(self, patient_id: str, session_id: str, scopes: list[str]) -> str:
            assert patient_id == "patient-1"
            assert session_id == "room-sid"
            assert "location:read" in scopes
            return "voice-scoped-token"

    userdata = SessionUserData(room_name="room", session_id="room-sid")
    userdata.session_auth = FakeAuth()
    userdata.welfare_check_context = {"run_id": "run-1", "patient_id": "patient-1"}
    userdata.upgrade("patient-1", "backend-token")

    _finalize_welfare_session_token(userdata)
    assert userdata.patient_token == "voice-scoped-token"


@pytest.mark.asyncio
async def test_report_welfare_run_status_calls_mcp_without_leaking_token_in_args_log() -> None:
    userdata = SessionUserData(room_name="room", session_id="sid")
    userdata.upgrade("patient-1", "secret-token")
    userdata.welfare_check_context = {
        "run_id": "run-1",
        "patient_id": "patient-1",
        "reason_code": "daily_checkup",
    }
    mcp = AsyncMock()
    await _report_welfare_run_status(userdata, mcp, "answered")
    mcp.call_tool.assert_awaited_once()
    args = mcp.call_tool.await_args.args
    assert args[0] == "update_welfare_run_status"
    payload = args[1]
    assert payload["status"] == "answered"
    assert payload["runID"] == "run-1"
    assert payload["_auth"] == "secret-token"
    assert "patient_token" not in payload
