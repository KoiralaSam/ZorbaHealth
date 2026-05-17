FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY pyproject.toml .
COPY src/ ./src/
RUN pip install --no-cache-dir uv && uv pip install --system . || \
    pip install --no-cache-dir \
        "livekit-agents[silero,turn-detector,openai,deepgram,elevenlabs]~=1.5" \
        "aiohttp>=3.9" \
        "PyJWT>=2.8" \
        "python-dotenv"

RUN python src/agent.py download-files

ENV PYTHONUNBUFFERED=1

CMD ["python", "src/agent.py", "start"]
