FROM golang:latest AS builder

WORKDIR /app

COPY notification ./notification
COPY pkg ./pkg

WORKDIR /app/notification

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./main/main.go


FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/notification/app .

CMD ["./app"]
