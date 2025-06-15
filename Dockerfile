FROM golang:1.24.3 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN /go/bin/swag init -g cmd/WeatherSubscriptionAPI/main.go

RUN CGO_ENABLED=0 go build -o weather-app ./cmd/WeatherSubscriptionAPI


FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/weather-app .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/web ./web
COPY --from=builder /app/docs ./docs

EXPOSE 8080

CMD ["./weather-app"]
