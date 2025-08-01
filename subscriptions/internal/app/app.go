package app

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/metrics"
	"github.com/rs/zerolog"

	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/producers"

	grpc2 "github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/handlers/grpc"
	http2 "github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/handlers/http"

	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/notifier"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/repository/sqlite"
	subs2 "github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/services/subscriptions"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/services/weather"

	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"

	"github.com/gin-gonic/gin"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerfiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

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
	SubscriptionService *subs2.Service
	EmailProducer       *producers.Producer
	Notificator         *notifier.Notifier
	SubRepository       sqlite.SubscriptionRepository
	GrpcServer          *grpc.Server

	Router     *gin.Engine
	Srv        *http.Server
	Db         *sql.DB
	fileLogger *zap.Logger
	M          *metrics.Metrics
}

type App struct {
	cfg config.Config
	l   zerolog.Logger
}

func New(cfg config.Config, logger zerolog.Logger) *App {
	// Enrich logger with service name and timestamp
	logger = logger.With().Str("service", "subscription-service").Timestamp().Logger()
	logger.Info().Msg("Logger initialized for subscription-service")
	return &App{cfg: cfg, l: logger}
}

func (a *App) Start(ctx context.Context) error {
	// Initialize metrics within Start to avoid changing App struct
	a.l.Info().Msg("Metrics initialized")

	srvContainer := a.init()
	a.l.Info().Str("grpc_port", a.cfg.Server.GrpcPort).Msg("Starting server")

	// Ensure HTTP server is closed on exit
	defer func() {
		if err := srvContainer.Srv.Close(); err != nil {
			a.l.Error().Err(err).Msg("Error closing HTTP server")
		}
	}()

	// Insert metrics middleware into router
	srvContainer.Router.Use(gin.Recovery(), func(c *gin.Context) {
		// Proxy to metrics middleware
		srvContainer.M.HTTPMiddleware()(c)
	})

	// Register HTTP endpoints
	subHandler := http2.NewHandler(srvContainer.SubscriptionService)

	srvContainer.Router.POST("/subscribe", subHandler.Subscribe)
	srvContainer.Router.GET("/confirm/:token", subHandler.Confirm)
	srvContainer.Router.GET("/unsubscribe/:token", subHandler.Unsubscribe)

	srvContainer.Router.GET("/swagger/*any", swagger.WrapHandler(swaggerfiles.Handler))
	srvContainer.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Start notifier
	ctxCron, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()
	srvContainer.Notificator.Start(ctxCron)
	a.l.Info().Msg("Notifier started")

	// Start HTTP server
	a.l.Info().Str("http_addr", a.cfg.ServerAddress()).Msg("HTTP server listening")
	if err := srvContainer.Srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		a.l.Error().Err(err).Msg("HTTP server error")
		return err
	}
	a.l.Info().Msg("HTTP server stopped")

	<-ctx.Done()
	a.l.Info().Msg("Shutdown signal received")
	return a.Stop(srvContainer)
}

func (a *App) Stop(srvContainer ServiceContainer) error {
	a.l.Info().Msg("Stopping application")

	// Stop notifier
	srvContainer.Notificator.Stop()
	a.l.Info().Msg("Notifier stopped")

	// Graceful gRPC shutdown
	srvContainer.GrpcServer.GracefulStop()
	a.l.Info().Msg("gRPC server stopped")

	// HTTP shutdown
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	if err := srvContainer.Srv.Shutdown(ctx); err != nil {
		a.l.Error().Err(err).Msg("HTTP shutdown error")
	} else {
		a.l.Info().Msg("HTTP server stopped")
	}

	// Close DB
	if err := srvContainer.Db.Close(); err != nil {
		a.l.Error().Err(err).Msg("Database close error")
	} else {
		a.l.Info().Msg("Database closed")
	}

	a.l.Info().Msg("Application shutdown complete")
	return nil
}

func (a *App) init() ServiceContainer {
	a.l.Info().Interface("config", a.cfg).Msg("Initializing application")

	// DB setup
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	db, err := CreateSqliteDb(ctx, a.cfg.DB.Dialect, a.cfg.DB.Source)
	if err != nil {
		a.l.Error().Err(err).Msg("DB open error")
	}
	if err := InitSqliteDb(db, a.cfg.DB.Dialect, a.cfg.DB.MigrationsPath); err != nil {
		a.l.Error().Err(err).Msg("DB migration error")
	}

	m := metrics.NewMetrics("subscription_service", db, a.cfg.DB.Source)

	// Gin router
	router := gin.New()

	// HTTP server
	httpSrv := &http.Server{
		Addr:        a.cfg.ServerAddress(),
		Handler:     router,
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout) * time.Second,
	}
	a.l.Info().Str("http_addr", a.cfg.ServerAddress()).Msg("HTTP server configured")

	// Repository
	repo := sqlite.NewSubscriptionRepository(db, a.l, m)

	// RabbitMQ
	rabbitConn, err := a.setupConn()
	if err != nil {
		a.l.Error().Err(err).Msg("RabbitMQ connection error")
	}
	publisher, err := a.setupPublisher(rabbitConn)
	if err != nil {
		a.l.Error().Err(err).Msg("RabbitMQ publisher error")
	}
	producer := producers.NewProducer(publisher, a.l, m)

	// Business services
	subSvc := subs2.NewService(repo, producer)
	grpcConn, err := grpc.DialContext(ctx, a.cfg.WeatherRPCAddr+a.cfg.WeatherRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		a.l.Error().Err(err).Msg("Weather gRPC dial error")
	}
	weatherClient := weatherpb.NewWeatherServiceClient(grpcConn)
	weatherSvc := weather.NewGrpcWeatherAdapter(weatherClient, a.l, m)

	// Notifier
	n := notifier.New(repo, weatherSvc, producer, a.l,
		a.cfg.NotifierFreq.HourlyFrequency,
		a.cfg.NotifierFreq.DailyFrequency,
		m,
	)

	// gRPC server with metrics interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(m.UnaryServerInterceptor()),
		grpc.StreamInterceptor(m.StreamServerInterceptor()),
	)
	subs.RegisterSubscriptionServiceServer(grpcServer, grpc2.NewSubscriptionGRPCServer(subSvc))

	// Start gRPC
	go func() {
		addr := a.cfg.Server.Host + ":" + a.cfg.Server.GrpcPort
		lc := net.ListenConfig{}
		lis, err := lc.Listen(ctx, "tcp", addr)
		if err != nil {
			a.l.Fatal().Err(err).Msg("gRPC listen error")
		}
		a.l.Info().Str("grpc_addr", addr).Msg("gRPC server running")
		if err := grpcServer.Serve(lis); err != nil {
			a.l.Error().Err(err).Msg("gRPC server error")
		}
	}()

	return ServiceContainer{
		WeatherService:      weatherSvc,
		SubscriptionService: subSvc,
		EmailProducer:       producer,
		Notificator:         n,
		SubRepository:       *repo,
		GrpcServer:          grpcServer,
		Router:              router,
		Srv:                 httpSrv,
		Db:                  db,
	}
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
