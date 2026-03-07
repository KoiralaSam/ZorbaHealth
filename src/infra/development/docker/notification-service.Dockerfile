FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY services/notification-service ./services/notification-service
COPY shared ./shared

# Build the application
WORKDIR /app/services/notification-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/notification-service ./cmd/notification-service

# Final stage
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/notification-service ./build/notification-service
COPY --from=builder /app/shared ./shared

ENTRYPOINT ["./build/notification-service"]
