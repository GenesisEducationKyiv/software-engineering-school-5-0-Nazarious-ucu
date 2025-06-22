package app

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/email"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/subscriptions"
	service "github.com/Nazarious-ucu/weather-subscription-api/internal/services/weather"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/subscription"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/weather"
	"github.com/pressly/goose/v3"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/emailer"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/notifier"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/repository"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
)

const timeoutDuration = 5 * time.Second

type App struct {
	cfg config.Config
	log *log.Logger
}

type ServiceContainer struct {
	weatherService      *service.Service
	subscriptionService *subscriptions.Service
	emailService        *email.Service
	notificator         *notifier.Notifier
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
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout) * time.Second,
	}

	smtpService := emailer.NewSMTPService(&a.cfg, a.log)
	subRepository := repository.NewSubscriptionRepository(db, a.log)
	emailService := email.NewService(smtpService, a.cfg.TemplatesDir)
	weatherService := service.NewService(a.cfg.WeatherAPIKey, &http.Client{}, a.log)
	notificator := notifier.New(subRepository,
		weatherService,
		emailService,
		a.log)

	srvContainer := ServiceContainer{
		weatherService:      weatherService,
		subscriptionService: subscriptions.NewService(subRepository, emailService),
		emailService:        emailService,
		subRepository:       *subRepository,
		notificator:         notificator,

		router: router,
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

	subHandler := subscription.NewHandler(srvContainer.subscriptionService)
	weatherHandler := weather.NewHandler(srvContainer.weatherService)

	notificator := notifier.New(&srvContainer.SubRepository,
		srvContainer.WeatherService, srvContainer.EmailService)

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

	log.Println("Application started successfully on", a.cfg.Server.Address)
	defer func() {
		if err := a.Stop(srvContainer); err != nil {
			log.Panicf("failed to shutdown application: %v", err)
		}
		log.Println("Application shutdown successfully")
	}()
	return nil
}

func (a *App) Stop(srvContainer ServiceContainer) error {
	a.log.Println("Stopping application…")

	srvContainer.notificator.Stop()
	a.log.Println("Notifier stopped")

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	if err := srvContainer.srv.Shutdown(ctx); err != nil {
		a.log.Println("HTTP shutdown error:", err)
	} else {
		a.log.Println("HTTP server stopped")
	}

	if err := srvContainer.db.Close(); err != nil {
		a.log.Println("DB close error:", err)
	} else {
		a.log.Println("Database closed")
	}

	a.log.Println("Shutdown complete")
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
