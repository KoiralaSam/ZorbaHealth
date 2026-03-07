FROM golang:1.24-alpine AS builder
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
COPY shared/go.mod shared/go.sum ./shared/

# Download dependencies
RUN go mod download

# Copy source code
COPY services/agent-worker-service ./services/agent-worker-service
COPY shared ./shared

# Build the application
WORKDIR /app/services/agent-worker-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/agent-worker-service ./cmd/agent-worker

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/build/agent-worker-service .
CMD ["./agent-worker-service"]
