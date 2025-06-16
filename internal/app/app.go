package app

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/emailer"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/notifier"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/repository"
	service "github.com/Nazarious-ucu/weather-subscription-api/internal/services"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type App struct {
	srv    *http.Server
	cfg    config.Config
	log    *log.Logger
	router *gin.Engine
	db     *sql.DB
}

func NewApp(cfg config.Config, logger *log.Logger, router *gin.Engine, db *sql.DB) *App {
	return &App{
		cfg:    cfg,
		log:    logger,
		router: router,
		db:     db,
	}
}

func (a *App) Init() error {
	a.log.Println("Initializing application with configuration:", a.cfg)

	a.srv = &http.Server{
		Addr:        a.cfg.Server.Address,
		Handler:     a.router,
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout),
	}
	subscriptionRepository := repository.NewSubscriptionRepository(a.db)
	smtpMailer := emailer.NewSMTPService(&a.cfg)
	emailService := service.NewEmailService(smtpMailer)
	subService := service.NewSubscriptionService(subscriptionRepository, emailService)
	weatherService := service.NewWeatherService(a.cfg.WeatherAPIKey)

	subHandler := handlers.NewSubscriptionHandler(subService)
	weatherHandler := handlers.NewWeatherHandler(weatherService)

	notificator := notifier.NewNotifier(subscriptionRepository, weatherService, emailService)

	api := a.router.Group("/api")
	{
		api.GET("/weather", weatherHandler.GetWeather)
		api.POST("/subscribe", subHandler.Subscribe)
		api.GET("/confirm/:token", subHandler.Confirm)
		api.GET("/unsubscribe/:token", subHandler.Unsubscribe)
	}
	a.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	notificator.StartWeatherNotifier()

	return nil
}

func (a *App) Start() error {
	a.log.Println("Starting server on", a.cfg.Server.Address)

	if err := a.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (a *App) Stop() error {
	a.log.Println("Stopping server on", a.cfg.Server.Address)

	// Graceful shutdown
	defer func(db *sql.DB) {
		if err := db.Close(); err != nil {
			log.Panicf("failed to close database connection: %v", err)
		}
	}(a.db)

	if err := a.srv.Close(); err != nil {
		a.log.Println("Error stopping server:", err)
		return err
	}
	a.log.Println("Server stopped successfully")
	return nil
}
