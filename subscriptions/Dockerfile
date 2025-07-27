FROM golang:latest AS builder

WORKDIR /app

COPY subscriptions ./subscriptions
COPY protos ./protos
COPY pkg ./pkg

WORKDIR /app/subscriptions

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o sub-app ./main/main.go


FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/subscriptions/sub-app .
COPY --from=builder /app/subscriptions/migrations ./migrations

EXPOSE 8080

CMD ["./sub-app"]
