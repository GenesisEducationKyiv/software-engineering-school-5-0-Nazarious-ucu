package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

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

	logger, err := zap.NewProduction()
	if err != nil {
		panic("cannot initialize zap logger: " + err.Error())
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			log.Printf("failed to sync logger: %v", err)
		} else {
			log.Println("logger synced successfully")
		}
	}(logger)

	sugar := logger.Sugar()

	application := app.New(*config, sugar)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Start(ctx); err != nil {
		log.Panic(err)
	}
}
