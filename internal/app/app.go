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
	WeatherService      *service.WeatherService
	SubscriptionService *service.SubscriptionService
	EmailService        *service.EmailService
	SubRepository       repository.SubscriptionRepository

	Router *gin.Engine
	Srv    *http.Server
	Db     *sql.DB
}

func New(cfg config.Config, logger *log.Logger) *App {
	return &App{
		cfg: cfg,
		log: logger,
	}
}

func (a *App) Init() ServiceContainer {
	a.log.Println("Initializing application with configuration:", a.cfg)

	db, err := CreateSqliteDb(a.cfg.DB.Dialect, a.cfg.DB.Source)
	if err != nil {
		log.Panic(err)
	}

	if err := InitSqliteDb(db, a.cfg.DB.Dialect, a.cfg.DB.MigrationsPath); err != nil {
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
	emailService := service.NewEmailService(smtpService, a.cfg.TemplatesDir)

	srvContainer := ServiceContainer{
		WeatherService:      service.NewWeatherService(a.cfg.WeatherAPIKey, &http.Client{}),
		SubscriptionService: service.NewSubscriptionService(subRepository, emailService),
		EmailService:        emailService,
		SubRepository:       *subRepository,

		Router: gin.Default(),
		Srv:    apiServer,
		Db:     db,
	}

	return srvContainer
}

func (a *App) Start(srvContainer ServiceContainer) error {
	a.log.Println("Starting server on", a.cfg.Server.Address)

	defer func() {
		if err := srvContainer.Srv.Close(); err != nil {
			a.log.Println("Error stopping server:", err)
		}
	}()

	subHandler := subscription.NewHandler(srvContainer.SubscriptionService)
	weatherHandler := weather.NewHandler(srvContainer.WeatherService)

	notificator := notifier.New(&srvContainer.SubRepository, srvContainer.WeatherService, srvContainer.EmailService)

	api := srvContainer.Router.Group("/api")
	{
		api.GET("/weather", weatherHandler.GetWeather)
		api.POST("/subscribe", subHandler.Subscribe)
		api.GET("/confirm/:token", subHandler.Confirm)
		api.GET("/unsubscribe/:token", subHandler.Unsubscribe)
	}
	srvContainer.Router.GET("/swagger/*any", swagger.WrapHandler(swaggerfiles.Handler))

	notificator.StartWeatherNotifier()

	if err := srvContainer.Srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	}(srvContainer.Db)

	a.log.Println("Server stopped successfully")
	return nil
}

func CreateSqliteDb(dialect, name string) (*sql.DB, error) {
	db, err := sql.Open(dialect, "file:weather.Db?cache=shared&mode=rwc")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func InitSqliteDb(db *sql.DB, dialect, migrationPath string) error {
	log.Println("Initializing migrations:", migrationPath)
	if err := goose.SetDialect(dialect); err != nil {
		return err
	}

	if err := goose.Up(db, migrationPath); err != nil {
		return err
	}

	return nil
}
