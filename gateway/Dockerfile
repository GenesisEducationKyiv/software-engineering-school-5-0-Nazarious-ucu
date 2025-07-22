FROM golang:latest AS builder

WORKDIR /app

COPY gateway/ ./gateway
COPY .env .env
COPY protos/ ./protos

WORKDIR /app/gateway

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gateway-app ./main/main.go

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY --from=builder /app/gateway .
COPY --from=builder /app/.env .env

ENV GIN_MODE=release

EXPOSE 8080

CMD ["./gateway-app"]
