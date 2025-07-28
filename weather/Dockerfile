FROM golang:latest as builder
LABEL authors="nazar"

WORKDIR /app

COPY weather/ ./weather
COPY .env .env
COPY protos/ ./protos
COPY docs/ ./docs


WORKDIR /app/weather
ENV WEATHER_SERVER_HOST="0.0.0.0"

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o weather-app ./main/main.go

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/weather/weather-app .
COPY --from=builder /app/docs ./docs

ENV WEATHER_SERVER_HOST="0.0.0.0"

EXPOSE 8080
CMD ["./weather-app"]