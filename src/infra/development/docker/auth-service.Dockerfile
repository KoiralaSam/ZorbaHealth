FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY services/auth-service ./services/auth-service
COPY shared ./shared

# Build the application
WORKDIR /app/services/auth-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/auth-service ./cmd/auth-service

# Final stage
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/auth-service ./build/auth-service
COPY --from=builder /app/shared ./shared

ENTRYPOINT ["./build/auth-service"]

