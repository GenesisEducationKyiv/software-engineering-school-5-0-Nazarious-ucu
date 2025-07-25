package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

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

	l := log.New(log.Writer(), "WeatherSubscriptionAPI: ", log.LstdFlags)

	// Initialize the application
	application := app.New(*cfg, l)

	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.Server.ReadTimeout)*time.Second)
	defer cancel()

	ctx, stop := signal.NotifyContext(ctxWithTimeout, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run the application
	if err := application.Start(ctx); err != nil {
		log.Panicf("Application failed to run: %v", err)
	}
}
