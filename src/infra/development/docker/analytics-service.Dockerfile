FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
COPY shared/go.mod shared/go.sum ./shared/

# Download dependencies
RUN go mod download

# Copy source code
COPY services/analytics-service ./services/analytics-service
COPY shared ./shared

# Build the application
WORKDIR /app/services/analytics-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/analytics-service ./cmd/analytics-service

# Final stage
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/analytics-service ./build/analytics-service
COPY --from=builder /app/shared ./shared

ENTRYPOINT ["./build/analytics-service"]
