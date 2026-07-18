# syntax=docker/dockerfile:1
FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libsndfile1 \
    && rm -rf /var/lib/apt/lists/*

COPY pyproject.toml .
COPY src/ ./src/

# pip only: avoids uv writing large temp caches under /tmp during image build.
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --no-cache-dir .

RUN python -m livekit.agents download-files

ENV PYTHONUNBUFFERED=1

CMD ["python", "src/agent.py", "start"]
