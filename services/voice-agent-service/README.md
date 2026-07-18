# Zorba Health Voice Agent Service

Python LiveKit Agents worker for Zorba Health voice calls.

Runtime flow:

```text
LiveKit SIP dispatch -> voice-agent-service -> mcp-server -> Go gRPC services
```

The service owns realtime audio, STT, LLM, TTS, VAD, and LiveKit job lifecycle.
Backend tools are called through the MCP server. Patient/session JWT creation
for the voice call lives in `src/auth`.

The service also exposes:

- `POST /webhook/livekit`
- `GET /health`

on `VOICE_AGENT_HTTP_ADDR` (default `:8090`) so LiveKit webhook callbacks can target
the same service that runs the active voice agent.

## Local Run

```bash
cp .env.example .env.local
uv sync
uv run python src/agent.py download-files
uv run python src/agent.py dev
```

Required environment:

- `LIVEKIT_URL`
- `LIVEKIT_API_KEY`
- `LIVEKIT_API_SECRET`
- `LIVEKIT_AGENT_NAME`
- `VOICE_AGENT_HTTP_ADDR`
- `MCP_SERVER_URL`
- `PATIENT_SERVICE_JWT_SECRET`
- `OPENAI_API_KEY`
- `DEEPGRAM_API_KEY`
- `ELEVENLABS_API_KEY`

## Tests

```bash
uv run pytest
```

## Kubernetes / worker tuning

When running `python src/agent.py start` (production worker mode), the LiveKit `AgentServer` reads:

| Variable | Default | Purpose |
|----------|---------|---------|
| `VOICE_AGENT_NUM_IDLE_PROCESSES` | `1` | Prewarmed job processes (faster SIP answer) |
| `VOICE_AGENT_LOAD_THRESHOLD` | `0.75` | Stop accepting jobs above this load (0–1) |
| `VOICE_AGENT_SHUTDOWN_PROCESS_TIMEOUT` | `30` | Grace period per job process on hangup |
| `VOICE_AGENT_INITIALIZE_PROCESS_TIMEOUT` | `45` | Time allowed for Silero VAD prewarm |
| `VOICE_AGENT_JOB_MEMORY_WARN_MB` | `1200` | Log warning if a job process exceeds this RSS |
| `VOICE_AGENT_JOB_MEMORY_LIMIT_MB` | `0` | Kill job process above this RSS (`0` = disabled) |

Set `ENABLE_TURN_DETECTOR=false` on small clusters; the multilingual turn model adds significant CPU/RAM during calls.

`VOICE_AGENT_REQUIRE_SIP_CALLER=true` (default) skips LiveKit jobs that have no SIP/phone participant so phantom room dispatches do not run the LLM greeting (~3k tokens). Set to `false` only for local `agent.py dev` without SIP.

SIP **trunk probe** rooms (`call-_trunk_*`, identity `sip_trunk_*`) are always skipped—they are not patient calls but still dispatch the agent and previously triggered the greeting LLM.

Health checks: `GET /health` on `VOICE_AGENT_HTTP_ADDR` (default `:8090`).
