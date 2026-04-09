FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/analytics-service ./services/analytics-service/cmd/analytics-service

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /app/build/analytics-service ./analytics-service

EXPOSE 50054

CMD ["./analytics-service"]
