import config


def test_require_sip_caller_defaults_true(monkeypatch) -> None:
    monkeypatch.delenv("VOICE_AGENT_REQUIRE_SIP_CALLER", raising=False)
    assert config._optional_bool("VOICE_AGENT_REQUIRE_SIP_CALLER", True) is True


def test_require_sip_caller_can_disable(monkeypatch) -> None:
    monkeypatch.setenv("VOICE_AGENT_REQUIRE_SIP_CALLER", "false")
    assert config._optional_bool("VOICE_AGENT_REQUIRE_SIP_CALLER", True) is False
