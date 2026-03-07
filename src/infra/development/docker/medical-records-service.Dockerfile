FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
COPY shared/go.mod shared/go.sum ./shared/

# Download dependencies
RUN go mod download

# Copy source code
COPY services/medical-records-service ./services/medical-records-service
COPY shared ./shared

# Build the application
WORKDIR /app/services/medical-records-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/medical-records-service ./cmd/medical-records-service

# Final stage
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/medical-records-service ./build/medical-records-service
COPY --from=builder /app/shared ./shared

ENTRYPOINT ["./build/medical-records-service"]
