FROM golang:1.24-alpine AS builder
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
COPY shared/go.mod shared/go.sum ./shared/

# Download dependencies
RUN go mod download

# Copy source code
COPY services/notification-service ./services/notification-service
COPY shared ./shared

# Build the application
WORKDIR /app/services/notification-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/notification-service ./cmd/notification-service

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/build/notification-service .
CMD ["./notification-service"]
