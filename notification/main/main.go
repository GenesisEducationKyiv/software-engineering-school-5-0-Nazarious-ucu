package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/metrics"
	"github.com/Nazarious-ucu/weather-subscription-api/pkg/logger"

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

	l, err := logger.NewLogger("log/notifications.log", "notification_service")
	if err != nil {
		log.Panicf("failed to initialize logger: %v", err)
	}

	m := metrics.NewMetrics("notification_service")

	application := app.New(*cfg, l, m)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Start(ctx); err != nil {
		log.Panic(err)
	}
}
