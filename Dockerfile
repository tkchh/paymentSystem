FROM golang:1.24-alpine AS builder


RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o payment-system \
    ./cmd/paymentSystem

FROM alpine:3.19

RUN adduser -D -u 10001 appuser

RUN apk add --no-cache sqlite

WORKDIR /app

COPY --from=builder /app/payment-system .

RUN mkdir -p /app/config
COPY config/config.example.yaml /app/config/config.yaml

RUN mkdir -p /app/data && chown -R appuser:appuser /app

USER appuser

ENV CONFIG_PATH=/app/config
ENV PAYMENT_STORAGE_PATH=/app/data/app.db

CMD ["./payment-system"]


