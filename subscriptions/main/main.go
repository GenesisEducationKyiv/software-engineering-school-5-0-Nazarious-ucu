package main

import (
	"context"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/metrics"
	"log"
	"os/signal"
	"syscall"

	"github.com/Nazarious-ucu/weather-subscription-api/pkg/logger"

	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/config"

	"github.com/joho/godotenv"
)

// @title Weather Subscription API
// @version 1.0
// @description API for subscribing to weather forecasts
// @host localhost:8080
// @BasePath /api/
func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	cfg, err := config.NewConfig()
	if err != nil {
		log.Panicf("failed to load configuration: %v", err)
	}

	l, err := logger.NewLogger("logs/subscriptions.log", "subscriptions")

	metr := metrics.NewMetrics("subscription")

	application := app.New(*cfg, l, metr)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Start(ctx); err != nil {
		log.Panic(err)
	}
}
