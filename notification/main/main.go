package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	cfg, err := config.NewConfig()
	if err != nil {
		log.Panicf("failed to load configuration: %v", err)
	}

	l := log.New(log.Writer(), "WeatherSubscriptionAPI: ", log.LstdFlags)

	application := app.New(*cfg, l)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Start(ctx); err != nil {
		log.Panic(err)
	}
}
