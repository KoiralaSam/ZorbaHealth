## syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY shared ./shared
COPY services/mcp-server ./services/mcp-server

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=linux go build -o /app/build/mcp-server ./services/mcp-server/cmd/mcp-server

FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/mcp-server /app/mcp-server

ENV MCP_TRANSPORT=http
ENV MCP_HTTP_ADDR=:8092

ENTRYPOINT ["/app/mcp-server"]
