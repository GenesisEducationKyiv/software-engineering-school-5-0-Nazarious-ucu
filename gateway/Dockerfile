FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o gateway ./main

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/gateway /app/gateway
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

COPY .env .env

ENV GIN_MODE=release

EXPOSE 8080

CMD ["./gateway"]
