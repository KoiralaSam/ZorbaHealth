## syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
# Cache modules between builds (BuildKit).
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy only what this service needs to build.
COPY shared ./shared
COPY services/translation-service ./services/translation-service

# Cache build artifacts between builds (BuildKit).
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -o /app/build/translation-service ./services/translation-service/cmd/translation-service

FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/translation-service /app/translation-service

ENTRYPOINT ["/app/translation-service"]
