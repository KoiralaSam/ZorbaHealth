import os

import config


def test_load_worker_config_defaults(monkeypatch):
    for key in (
        "VOICE_AGENT_NUM_IDLE_PROCESSES",
        "VOICE_AGENT_LOAD_THRESHOLD",
        "VOICE_AGENT_SHUTDOWN_PROCESS_TIMEOUT",
        "VOICE_AGENT_INITIALIZE_PROCESS_TIMEOUT",
        "VOICE_AGENT_JOB_MEMORY_WARN_MB",
        "VOICE_AGENT_JOB_MEMORY_LIMIT_MB",
    ):
        monkeypatch.delenv(key, raising=False)

    wc = config.load_worker_config()
    assert wc.num_idle_processes == 1
    assert wc.load_threshold == 0.75
    assert wc.shutdown_process_timeout == 30.0
    assert wc.initialize_process_timeout == 45.0


def test_load_worker_config_overrides(monkeypatch):
    monkeypatch.setenv("VOICE_AGENT_NUM_IDLE_PROCESSES", "2")
    monkeypatch.setenv("VOICE_AGENT_LOAD_THRESHOLD", "0.8")
    wc = config.load_worker_config()
    assert wc.num_idle_processes == 2
    assert wc.load_threshold == 0.8
