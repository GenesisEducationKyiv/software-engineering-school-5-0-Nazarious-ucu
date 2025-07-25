package main

import (
	"context"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/config"
	"github.com/joho/godotenv"
	"log"
	"os/signal"
	"syscall"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run the application
	if err := application.Start(ctx); err != nil {
		log.Panicf("Application failed to run: %v", err)
	}
}
