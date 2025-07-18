package app

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"

	"github.com/gin-gonic/gin"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerfiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "github.com/Nazarious-ucu/weather-subscription-api/docs"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/emailer"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/subscription"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/weather"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/notifier"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/repository"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/cache"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/email"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/logger"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/metrics"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/subscriptions"
	serviceWeather "github.com/Nazarious-ucu/weather-subscription-api/internal/services/weather"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/weather/decorators"
	fLogger "github.com/Nazarious-ucu/weather-subscription-api/pkg/logger"
	"github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/subs"
	weatherpb "github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/weather"
)

const (
	timeoutDuration = 5 * time.Second
)

type weatherGetterService interface {
	GetByCity(ctx context.Context, city string) (models.WeatherData, error)
}

type ServiceContainer struct {
	WeatherService      weatherGetterService
	SubscriptionService *subscriptions.Service
	EmailService        *email.Service
	Notificator         *notifier.Notifier
	SubRepository       repository.SubscriptionRepository
	GrpcServer          *grpc.Server

	Router     *gin.Engine
	Srv        *http.Server
	Db         *sql.DB
	fileLogger *zap.Logger
}

type App struct {
	cfg config.Config
	log *log.Logger
}

func New(cfg config.Config, logger *log.Logger) *App {
	return &App{
		cfg: cfg,
		log: logger,
	}
}

func (a *App) Start(ctx context.Context) error {
	srvContainer := a.init()
	a.log.Println("Starting server on", a.cfg.Server.Address)

	defer func() {
		if err := srvContainer.Srv.Close(); err != nil {
			a.log.Println("Error stopping server:", err)
		}
	}()

	subHandler := subscription.NewHandler(srvContainer.SubscriptionService)
	weatherHandler := weather.NewHandler(srvContainer.WeatherService)

	api := srvContainer.Router.Group("/api")
	{
		api.GET("/weather", weatherHandler.GetWeather)
		api.POST("/subscribe", subHandler.Subscribe)
		api.GET("/confirm/:token", subHandler.Confirm)
		api.GET("/unsubscribe/:token", subHandler.Unsubscribe)
	}
	srvContainer.Router.GET("/swagger/*any", swagger.WrapHandler(swaggerfiles.Handler))
	srvContainer.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	srvContainer.Notificator.Start(ctxWithTimeout)

	if err := srvContainer.Srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	<-ctx.Done()

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

	defer func(fileLogger *zap.Logger) {
		err := fileLogger.Sync()
		if err != nil {
			a.log.Printf("failed to sync file logger: %v", err)
		} else {
			a.log.Println("File logger synced successfully")
		}
	}(srvContainer.fileLogger)

	srvContainer.Notificator.Stop()
	a.log.Println("Notifier stopped")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gRPC server...")
	srvContainer.GrpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	if err := srvContainer.Srv.Shutdown(ctx); err != nil {
		a.log.Println("HTTP shutdown error:", err)
	} else {
		a.log.Println("HTTP server stopped")
	}

	if err := srvContainer.Db.Close(); err != nil {
		a.log.Println("DB close error:", err)
	} else {
		a.log.Println("Database closed")
	}

	a.log.Println("Shutdown complete")
	return nil
}

func (a *App) init() ServiceContainer {
	a.log.Println("Initializing application with configuration:", a.cfg)

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	db, err := CreateSqliteDb(ctx, a.cfg.DB.Dialect, a.cfg.DB.Source)
	if err != nil {
		a.log.Panic(err)
	}

	if err := InitSqliteDb(db, a.cfg.DB.Dialect, a.cfg.DB.MigrationsPath); err != nil {
		a.log.Panic(err)
	}

	redisClient := newRedisConnection(a.cfg.Redis.Host+":"+a.cfg.Redis.Port, a.cfg.Redis.DbType)
	router := gin.Default()

	apiServer := &http.Server{
		Addr:        a.cfg.Server.Address,
		Handler:     router,
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout) * time.Second,
	}

	a.log.Printf("Application initialized on address: %s", apiServer.Addr)

	smtpService := emailer.NewSMTPService(&a.cfg, a.log)
	a.log.Printf("Initializing SMTP service with config: %+v\n", a.cfg.Email)
	subRepository := repository.NewSubscriptionRepository(db, a.log)
	emailService := email.NewService(smtpService, a.cfg.TemplatesDir)

	fileLogger, err := fLogger.NewFileLogger(a.cfg.LogsPath)
	if err != nil {
		a.log.Panicf("failed to create file logger: %v", err)
	}

	loggerT := logger.NewRoundTripper(fileLogger)

	httpLogClient := &http.Client{
		Transport: loggerT,
	}

	breakerCfg := serviceWeather.BreakerConfig{
		TimeInterval: time.Duration(a.cfg.Breaker.TimeInterval) * time.Second,
		TimeTimeOut:  time.Duration(a.cfg.Breaker.TimeTimeOut) * time.Second,
		RepeatNumber: a.cfg.Breaker.RepeatNumber,
	}
	openWeatherMapClient := serviceWeather.NewBreakerClient("OpenWeather", breakerCfg,
		serviceWeather.NewClientOpenWeatherMap(
			a.cfg.OpenWeatherMapAPIKey,
			a.cfg.OpenWeatherMapURL,
			httpLogClient,
			a.log,
		),
	)

	weatherAPIClient := serviceWeather.NewBreakerClient("WeatherAPI", breakerCfg,
		serviceWeather.NewClientWeatherAPI(
			a.cfg.WeatherAPIKey,
			a.cfg.WeatherAPIURL,
			httpLogClient,
			a.log,
		),
	)

	weatherBitClient := serviceWeather.NewBreakerClient("WeatherBit", breakerCfg,
		serviceWeather.NewClientWeatherBit(
			a.cfg.WeatherBitAPIKey,
			a.cfg.WeatherBitURL,
			httpLogClient,
			a.log,
		),
	)

	weatherService := serviceWeather.NewService(a.log,
		weatherAPIClient,
		openWeatherMapClient,
		weatherBitClient)

	prom := metrics.NewPromCollector()

	cacheRedisClient := cache.NewRedisClient[models.WeatherData](
		redisClient,
		a.log,
		time.Duration(a.cfg.Redis.LiveTime)*time.Hour,
	)
	cacheWithMetrics := cache.NewMetricsDecorator[models.WeatherData](cacheRedisClient, prom)

	cacheDecorator := decorators.NewCachedService(weatherService, cacheWithMetrics, a.log)

	subService := subscriptions.NewService(subRepository, emailService)

	lis, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil {
		a.log.Panic(err)
	}
	grpcServer := grpc.NewServer()

	opt := grpc.WithTransportCredentials(insecure.NewCredentials())
	grpcClient, err := grpc.NewClient(a.cfg.WeatherRPCAddr+a.cfg.WeatherRPCPort, opt)

	if err != nil {
		a.log.Panicf("failed to create gRPC client: %v", err)
	} else {
		a.log.Printf(
			"gRPC client created successfully for address: %s",
			a.cfg.WeatherRPCAddr+a.cfg.WeatherRPCPort,
		)
	}

	weatherGrpc := weatherpb.NewWeatherServiceClient(grpcClient)

	weatherAdapter := serviceWeather.NewGrpcWeatherAdapter(weatherGrpc, a.log)

	weatherpb.RegisterWeatherServiceServer(grpcServer, decorators.NewWeatherGRPCServer(weatherAdapter))
	subs.RegisterSubscriptionServiceServer(grpcServer, subscriptions.NewSubscriptionGRPCServer(subService))

	go func() {
		log.Println("gRPC server running at :50051")
		if err := grpcServer.Serve(lis); err != nil {
			a.log.Panicf("gRPC server failed: %v", err)
		}
	}()

	notificator := notifier.New(subRepository,
		cacheDecorator,
		emailService,
		a.log,
		a.cfg.NotifierFreq.HourlyFrequency,
		a.cfg.NotifierFreq.DailyFrequency,
	)

	srvContainer := ServiceContainer{
		WeatherService:      *weatherAdapter,
		SubscriptionService: subService,
		EmailService:        emailService,
		SubRepository:       *subRepository,
		Notificator:         notificator,
		GrpcServer:          grpcServer,

		Router: router,
		Srv:    apiServer,
		Db:     db,
	}

	return srvContainer
}

func CreateSqliteDb(ctx context.Context, dialect, name string) (*sql.DB, error) {
	if name == "" {
		return nil, errors.New("database name cannot be empty")
	}
	connectionString := "file:" + name + "?cache=shared&mode=rwc"
	db, err := sql.Open(dialect, connectionString)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
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

func newRedisConnection(connString string, dbType int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: connString,
		DB:   dbType,
	})
}
