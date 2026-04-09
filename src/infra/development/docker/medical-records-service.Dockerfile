## syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY services/medical-records-service ./services/medical-records-service
COPY shared ./shared

# Build the application
WORKDIR /app/services/medical-records-service
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -o /app/build/medical-records-service ./cmd/medical-records-service

# Final stage
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/medical-records-service ./build/medical-records-service
COPY --from=builder /app/shared ./shared

ENTRYPOINT ["./build/medical-records-service"]
