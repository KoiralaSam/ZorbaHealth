## syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY services/health-provider-service ./services/health-provider-service
COPY shared ./shared

WORKDIR /app/services/health-provider-service
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/build/health-provider-service ./cmd/health-provider-service

FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/health-provider-service ./build/health-provider-service
COPY --from=builder /app/shared ./shared

ENTRYPOINT ["./build/health-provider-service"]
