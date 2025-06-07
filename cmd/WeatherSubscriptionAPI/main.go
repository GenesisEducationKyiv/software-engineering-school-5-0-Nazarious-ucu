package main

import (
	"WeatherSubscriptionAPI/internal/handlers"
	"WeatherSubscriptionAPI/internal/notifier"
	"WeatherSubscriptionAPI/internal/repository"
	service "WeatherSubscriptionAPI/internal/services"
	"database/sql"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "modernc.org/sqlite"
)

// @title Weather Subscription API
// @version 1.0
// @description API for subscribing to weather forecasts
// @host localhost:8080
// @BasePath /api/
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  Warning: .env file not found or failed to load")
	}

	db, err := sql.Open("sqlite", "./subscriptions.db")
	if err != nil {
		log.Panic(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Panic(err)
		}
	}(db)

	if err := goose.SetDialect("sqlite"); err != nil {
		log.Panic(err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		log.Panic(err)
	}

	r := gin.Default()

	r.StaticFile("/", "./web/index.html")
	r.Static("/web", "./web")

	repo := repository.NewSubscriptionRepository(db)
	emailService := service.NewEmailService()
	subService := service.NewSubscriptionService(repo, emailService)
	weatherService := &service.WeatherService{APIKey: os.Getenv("WEATHER_API_KEY")}
	subHandler := handlers.NewSubscriptionHandler(subService)
	weatherHandler := handlers.NewWeatherHandler(weatherService)

	api := r.Group("/api")
	{
		api.GET("/weather", weatherHandler.GetWeather)
		api.POST("/subscribe", subHandler.Subscribe)
		api.GET("/confirm/:token", subHandler.Confirm)
		api.GET("/unsubscribe/:token", subHandler.Unsubscribe)
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	notifier.StartWeatherNotifier(repo, weatherService, emailService)
	err = r.Run(":8080")
	if err != nil {
		return
	}
}
