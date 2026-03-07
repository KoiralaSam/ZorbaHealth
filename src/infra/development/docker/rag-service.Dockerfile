FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
COPY shared/go.mod shared/go.sum ./shared/

# Download dependencies
RUN go mod download

# Copy source code
COPY services/rag-service ./services/rag-service
COPY shared ./shared

# Build the application
WORKDIR /app/services/rag-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/rag-service ./cmd/rag-service

# Final stage
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/rag-service ./build/rag-service
COPY --from=builder /app/shared ./shared

ENTRYPOINT ["./build/rag-service"]
