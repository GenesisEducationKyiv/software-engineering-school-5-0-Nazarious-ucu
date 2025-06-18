package app

import (
	"database/sql"
	"errors"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/subscription"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/weather"
	"github.com/pressly/goose/v3"
	"log"
	"net/http"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/emailer"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/notifier"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/repository"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/services"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
)

type App struct {
	cfg config.Config
	log *log.Logger
}

type ServiceContainer struct {
	weatherService      *service.WeatherService
	subscriptionService *service.SubscriptionService
	emailService        *service.EmailService
	subRepository       repository.SubscriptionRepository

	router *gin.Engine
	srv    *http.Server
	db     *sql.DB
}

func New(cfg config.Config, logger *log.Logger) *App {
	return &App{
		cfg: cfg,
		log: logger,
	}
}

func (a *App) Init() ServiceContainer {
	a.log.Println("Initializing application with configuration:", a.cfg)

	db, err := createSqliteDb()
	if err != nil {
		log.Panic(err)
	}

	if err := initSqliteDb(db); err != nil {
		log.Panic(err)
	}

	router := gin.Default()

	apiServer := &http.Server{
		Addr:        a.cfg.Server.Address,
		Handler:     router,
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout),
	}

	smtpService := emailer.NewSMTPService(&a.cfg)
	subRepository := repository.NewSubscriptionRepository(db)
	emailService := service.NewEmailService(smtpService)

	srvContainer := ServiceContainer{
		weatherService:      service.NewWeatherService(a.cfg.WeatherAPIKey, &http.Client{}),
		subscriptionService: service.NewSubscriptionService(subRepository, emailService),
		emailService:        emailService,
		subRepository:       *subRepository,

		router: gin.Default(),
		srv:    apiServer,
		db:     db,
	}

	return srvContainer
}

func (a *App) Start(srvContainer ServiceContainer) error {
	a.log.Println("Starting server on", a.cfg.Server.Address)

	defer func() {
		if err := srvContainer.srv.Close(); err != nil {
			a.log.Println("Error stopping server:", err)
		}
	}()

	subHandler := subscription.NewSubscriptionHandler(srvContainer.subscriptionService)
	weatherHandler := weather.NewHandler(srvContainer.weatherService)

	notificator := notifier.New(&srvContainer.subRepository, srvContainer.weatherService, srvContainer.emailService)

	api := srvContainer.router.Group("/api")
	{
		api.GET("/weather", weatherHandler.GetWeather)
		api.POST("/subscribe", subHandler.Subscribe)
		api.GET("/confirm/:token", subHandler.Confirm)
		api.GET("/unsubscribe/:token", subHandler.Unsubscribe)
	}
	srvContainer.router.GET("/swagger/*any", swagger.WrapHandler(swaggerfiles.Handler))

	notificator.StartWeatherNotifier()

	if err := srvContainer.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (a *App) Stop(srvContainer ServiceContainer) error {
	a.log.Println("Stopping server on", a.cfg.Server.Address)

	// Graceful shutdown
	defer func(db *sql.DB) {
		if err := db.Close(); err != nil {
			log.Panicf("failed to close database connection: %v", err)
		}
	}(srvContainer.db)

	a.log.Println("Server stopped successfully")
	return nil
}

func createSqliteDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:weather.db?cache=shared&mode=rwc")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func initSqliteDb(db *sql.DB) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	if err := goose.Up(db, "./migrations"); err != nil {
		return err
	}

	return nil
}
