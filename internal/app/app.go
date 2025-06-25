package app

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/email"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/subscriptions"
	serviceWeather "github.com/Nazarious-ucu/weather-subscription-api/internal/services/weather"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/subscription"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/weather"
	"github.com/pressly/goose/v3"

	_ "github.com/Nazarious-ucu/weather-subscription-api/docs"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/emailer"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/notifier"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/repository"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
)

const (
	timeoutDuration = 5 * time.Second

	fileMode = 0o644
)

type LoggerRoundTripper interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

type App struct {
	cfg config.Config
	log *log.Logger
}

type ServiceContainer struct {
	weatherService      *serviceWeather.ServiceProvider
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

	db, err := createSqliteDb()
	if err != nil {
		a.log.Panic(err)
	}

	if err := initSqliteDb(db); err != nil {
		a.log.Panic(err)
	}

	router := gin.Default()

	apiServer := &http.Server{
		Addr:        a.cfg.Server.Address,
		Handler:     router,
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout) * time.Second,
	}

	smtpService := emailer.NewSMTPService(&a.cfg, a.log)
	subRepository := repository.NewSubscriptionRepository(db, a.log)
	emailService := email.NewService(smtpService)

	fileLogger, err := NewFileLogger(a.cfg.LogsPath)
	if err != nil {
		a.log.Panicf("failed to create file logger: %v", err)
	}
	defer func(fileLogger *zap.Logger) {
		err := fileLogger.Sync()
		if err != nil {
			a.log.Printf("failed to sync file logger: %v", err)
		} else {
			a.log.Println("File logger synced successfully")
		}
	}(fileLogger)

	loggerT := logger.NewRoundTripper(fileLogger)

	httpLogClient := &http.Client{
		Transport: loggerT,
	}
	openWeatherMapClient := serviceWeather.NewOpenWeatherMapClient(
		a.cfg.OpenWeatherMapAPIKey,
		httpLogClient,
		a.log,
	)

	weatherAPIClient := serviceWeather.NewWeatherAPIClient(a.cfg.WeatherAPIKey, httpLogClient, a.log)

	weatherBitClient := serviceWeather.NewWeatherBitClient(a.cfg.WeatherBitAPIKey, httpLogClient, a.log)

	weatherService := serviceWeather.NewService(a.log,
		weatherAPIClient,
		openWeatherMapClient,
		weatherBitClient)
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

	api := srvContainer.router.Group("/api")
	{
		api.GET("/weather", weatherHandler.GetWeather)
		api.POST("/subscribe", subHandler.Subscribe)
		api.GET("/confirm/:token", subHandler.Confirm)
		api.GET("/unsubscribe/:token", subHandler.Unsubscribe)
	}
	srvContainer.router.GET("/swagger/*any", swagger.WrapHandler(swaggerfiles.Handler))

	srvContainer.notificator.Start(context.Background())

	if err := srvContainer.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	a.log.Println("Application started successfully on", a.cfg.Server.Address)
	defer func() {
		if err := a.Stop(srvContainer); err != nil {
			a.log.Panicf("failed to shutdown application: %v", err)
		}
		a.log.Println("Application shutdown successfully")
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

func NewFileLogger(filePath string) (*zap.Logger, error) {
	file, err := os.OpenFile(filepath.Clean(filePath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileMode)
	if err != nil {
		return nil, err
	}

	writer := zapcore.AddSync(file)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		writer,
		zap.InfoLevel,
	)
	return zap.New(core), nil
}
