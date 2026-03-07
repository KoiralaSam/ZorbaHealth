FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY services/patient-service ./services/patient-service
COPY shared ./shared

# Build the application
WORKDIR /app/services/patient-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/patient-service ./cmd/patient-service

# Final stage
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/patient-service ./build/patient-service
COPY --from=builder /app/shared ./shared

ENTRYPOINT ["./build/patient-service"]
