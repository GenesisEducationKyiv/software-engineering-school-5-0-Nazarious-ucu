package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/metrics"

	"github.com/Nazarious-ucu/weather-subscription-api/pkg/logger"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	cfg, err := config.NewConfig()
	if err != nil {
		log.Panicf("failed to load configuration: %v", err)
	}

	l, err := logger.NewLogger(cfg.LogsPath, "weather")
	if err != nil {
		panic("cannot initialize logger: " + err.Error())
	}

	m := metrics.NewMetrics("weather_service")

	// Initialize the application
	application := app.New(*cfg, l, m)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run the application
	if err := application.Start(ctx); err != nil {
		log.Panicf("Application failed to run: %v", err)
	}
}
