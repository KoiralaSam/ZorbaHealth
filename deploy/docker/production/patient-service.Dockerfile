FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
COPY shared/go.mod shared/go.sum ./shared/

# Download dependencies
RUN go mod download

# Copy source code
COPY services/patient-service ./services/patient-service
COPY shared ./shared

# Build the application
WORKDIR /app/services/patient-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/patient-service ./cmd/patient-service && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/build/cron-dispatcher ./cmd/cron-dispatcher

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/build/patient-service .
COPY --from=builder /app/build/cron-dispatcher ./cron-dispatcher
CMD ["./patient-service"]
