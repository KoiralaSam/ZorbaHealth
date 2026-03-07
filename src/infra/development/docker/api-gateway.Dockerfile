FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY services/api-gateway ./services/api-gateway
COPY shared ./shared

# Build the application
WORKDIR /app/services/api-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/api-gateway ./cmd/api-gateway

# Final stage
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/api-gateway ./build/api-gateway
COPY --from=builder /app/shared ./shared

ENTRYPOINT ["./build/api-gateway"]
