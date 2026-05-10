## syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk --no-cache add build-base pkgconf opus-dev opusfile-dev soxr-dev

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY shared ./shared
COPY services/agent-worker-service ./services/agent-worker-service
COPY services/mcp-server ./services/mcp-server

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=linux go build -o /app/build/agent-worker-service ./services/agent-worker-service/cmd/agent-worker
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=linux go build -o /app/build/mcp-server ./services/mcp-server/cmd/mcp-server

FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates opus opusfile soxr

COPY --from=builder /app/build/agent-worker-service /app/agent-worker-service
COPY --from=builder /app/build/mcp-server /app/mcp-server

ENV MCP_SERVER_BINARY=/app/mcp-server

ENTRYPOINT ["/app/agent-worker-service"]
