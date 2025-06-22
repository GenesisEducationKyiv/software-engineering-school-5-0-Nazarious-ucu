package main

import (
	"log"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
)

// @title Weather Subscription API
// @version 1.0
// @description API for subscribing to weather forecasts
// @host localhost:8080
// @BasePath /api/
func main() {
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

	log.Println("Application started successfully on", cfg.Server.Address)
	defer func() {
		if err := application.Stop(serviceContainer); err != nil {
			log.Panicf("failed to shutdown application: %v", err)
		}
		log.Println("Application shutdown successfully")
	}()
}
