from agent import (
    _is_actionable_caller_identity,
    _is_trunk_probe_room,
    _skip_session_reason,
)


def test_is_actionable_caller_identity() -> None:
    assert _is_actionable_caller_identity("sip_+13185551212")
    assert _is_actionable_caller_identity("sip_66660113419814782")
    assert _is_actionable_caller_identity("+13185551212")
    assert not _is_actionable_caller_identity("sip_trunk_9")
    assert not _is_actionable_caller_identity("zorba-health-voice")
    assert not _is_actionable_caller_identity("short")


def test_trunk_probe_room_names() -> None:
    assert _is_trunk_probe_room("call-_trunk_9_QJNHuddasnMo")
    assert not _is_trunk_probe_room("call-_66660113419814782_RcA3yoRViTGa")


def test_skip_session_reason_trunk_probe() -> None:
    assert (
        _skip_session_reason(
            room_name="call-_trunk_9_QJNHuddasnMo",
            caller_identity="sip_trunk_9",
            caller_phone="9",
            require_sip_caller=True,
            min_caller_phone_digits=10,
        )
        == "SIP trunk probe room (not a patient call)"
    )


def test_skip_session_reason_short_phone() -> None:
    assert (
        _skip_session_reason(
            room_name="call-_foo",
            caller_identity="sip_trunk_9",
            caller_phone="9",
            require_sip_caller=True,
            min_caller_phone_digits=10,
        )
        is not None
    )


def test_skip_session_reason_valid_call() -> None:
    assert (
        _skip_session_reason(
            room_name="call-_66660113419814782_RcA3yoRViTGa",
            caller_identity="sip_66660113419814782",
            caller_phone="66660113419814782",
            require_sip_caller=True,
            min_caller_phone_digits=10,
        )
        is None
    )
