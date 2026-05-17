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
