## syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY shared ./shared
COPY services/audit-service ./services/audit-service

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/build/audit-service ./services/audit-service/cmd/audit-service

FROM alpine:3.19
WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/audit-service ./audit-service

EXPOSE 50058

CMD ["./audit-service"]
