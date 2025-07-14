package main

import (
	"context"
	"log"

	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/cfg"
	"github.com/joho/godotenv"
)

// @title Weather Subscription API
// @version 1.0
// @description API for subscribing to weather forecasts
// @host localhost:8080
// @BasePath /api/
func main() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	config, err := cfg.NewConfig()
	if err != nil {
		log.Panicf("failed to load configuration: %v", err)
	}

	l := log.New(log.Writer(), "WeatherSubscriptionAPI: ", log.LstdFlags)

	application := app.New(*config, l)

	if err := application.Start(context.Background()); err != nil {
		log.Panic(err)
	}
}
