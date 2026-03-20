FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY services/health-records-service ./services/health-records-service
COPY shared ./shared

# Build the application
WORKDIR /app/services/health-records-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/health-records-service ./cmd/medical-records-service

# Final stage
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/health-records-service ./build/health-records-service
COPY --from=builder /app/shared ./shared

ENTRYPOINT ["./build/health-records-service"]

