## syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY services/api-gateway ./services/api-gateway
COPY shared ./shared

# Build the application
WORKDIR /app/services/api-gateway
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -o /app/build/api-gateway ./cmd/api-gateway

# Final stage
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/api-gateway ./build/api-gateway
COPY --from=builder /app/shared ./shared

ENTRYPOINT ["./build/api-gateway"]
