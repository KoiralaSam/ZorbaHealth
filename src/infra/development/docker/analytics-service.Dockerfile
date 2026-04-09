## syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY shared ./shared
COPY services/analytics-service ./services/analytics-service

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -o /app/build/analytics-service ./services/analytics-service/cmd/analytics-service

FROM alpine:3.19
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/analytics-service ./analytics-service

EXPOSE 50054

CMD ["./analytics-service"]
