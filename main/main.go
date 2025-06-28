package main

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
)

// @title Weather Subscription API
// @version 1.0
// @description API for subscribing to weather forecasts
// @host localhost:8080
// @BasePath /api/
func main() {
	if err := godotenv.Load(".env.sample"); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	cfg, err := config.NewConfig()
	if err != nil {
		log.Panicf("failed to load configuration: %v", err)
	}

	logger := log.New(log.Writer(), "WeatherSubscriptionAPI: ", log.LstdFlags)

	application := app.New(*cfg, logger)

	serviceContainer := application.Init()

	if err := application.Start(serviceContainer); err != nil {
		log.Panic(err)
	}
}
