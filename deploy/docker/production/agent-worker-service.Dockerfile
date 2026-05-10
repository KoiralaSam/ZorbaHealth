FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk --no-cache add build-base pkgconf opus-dev opusfile-dev soxr-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=linux go build -o /app/build/agent-worker-service ./services/agent-worker-service/cmd/agent-worker
RUN GOOS=linux go build -o /app/build/mcp-server ./services/mcp-server/cmd/mcp-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates opus opusfile soxr
WORKDIR /app

COPY --from=builder /app/build/agent-worker-service /app/agent-worker-service
COPY --from=builder /app/build/mcp-server /app/mcp-server

ENV MCP_SERVER_BINARY=/app/mcp-server

CMD ["/app/agent-worker-service"]
