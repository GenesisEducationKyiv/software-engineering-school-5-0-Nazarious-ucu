package app

import (
	"context"
	"net"
	"net/http"
	"time"

	grpc2 "github.com/Nazarious-ucu/weather-subscription-api/weather/internal/handlers/grpc"
	http2 "github.com/Nazarious-ucu/weather-subscription-api/weather/internal/handlers/http"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/weather"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/cache"
	loggerT "github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/logger"
	metricsSvc "github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/metrics"
	serviceWeather "github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/weather"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/weather/decorators"
	fLogger "github.com/Nazarious-ucu/weather-subscription-api/weather/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// ServiceContainer holds initialized dependencies for servers.
type ServiceContainer struct {
	WeatherService *decorators.CachedService
	GrpcServer     *grpc.Server

	Router     *gin.Engine
	Srv        *http.Server
	fileLogger *zap.Logger
}

// App ties together config, logger, and metrics for startup/shutdown.
type App struct {
	cfg config.Config
	l   zerolog.Logger
	m   *metricsSvc.Metrics
}

// New prepares a new App with given config, zerolog logger, and metrics.
func New(cfg config.Config, logger zerolog.Logger, met *metricsSvc.Metrics) *App {
	return &App{
		cfg: cfg,
		l:   logger,
		m:   met,
	}
}

// Start initializes services, applies logging & metrics middleware, and waits for shutdown.
func (a *App) Start(ctx context.Context) error {
	srvContainer := a.init(ctx)

	a.l.Info().
		Str("grpc_port", a.cfg.Server.GrpcPort).
		Msg("starting weather service")

	// HTTP Metrics endpoint via Gin
	srvContainer.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Apply HTTP metrics and logging middleware
	srvContainer.Router.Use(a.m.HTTPMiddleware())

	// Mount weather HTTP endpoint
	weatherHandler := http2.NewHandler(srvContainer.WeatherService)
	srvContainer.Router.GET("/weather", weatherHandler.GetWeather)

	a.l.Info().
		Str("grpc_port", a.cfg.Server.GrpcPort).
		Msg("weather service started successfully")

	<-ctx.Done()
	a.l.Info().Msg("shutdown signal received, stopping weather service")

	if err := a.Shutdown(srvContainer); err != nil {
		a.l.Error().Err(err).Msg("failed to shutdown application")
		return err
	}
	a.l.Info().Msg("application shutdown successfully")
	return nil
}

// Shutdown performs graceful shutdown of gRPC and syncs loggers.
func (a *App) Shutdown(srvContainer ServiceContainer) error {
	a.l.Info().Msg("stopping weather service…")

	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			a.l.Error().Err(err).Msg("failed to sync file logger")
		} else {
			a.l.Info().Msg("file logger synced successfully")
		}
	}(srvContainer.fileLogger)

	a.l.Info().Msg("shutting down gRPC server")
	srvContainer.GrpcServer.GracefulStop()
	a.l.Info().Msg("shutdown complete")
	return nil
}

// init sets up logging, caching, metrics, HTTP & gRPC servers without starting them.
func (a *App) init(ctx context.Context) ServiceContainer {
	a.l.Info().Msgf("initializing weather service with config: %+v", a.cfg)

	// Redis cache client + metrics decorator
	redisClient := newRedisConnection(a.cfg.Redis.Host+":"+a.cfg.Redis.Port, a.cfg.Redis.DbType)

	fileLogger, err := fLogger.NewFileLogger(a.cfg.LogsPath)
	if err != nil {
		a.l.Error().Err(err).Msg("failed to create file logger")
	}

	// HTTP client logging
	roundTripper := loggerT.NewRoundTripper(fileLogger)
	httpLogClient := &http.Client{Transport: roundTripper}

	// Weather service with circuit breakers
	breakerCfg := serviceWeather.BreakerConfig{
		TimeInterval: time.Duration(a.cfg.Breaker.TimeInterval) * time.Second,
		TimeTimeOut:  time.Duration(a.cfg.Breaker.TimeTimeOut) * time.Second,
		RepeatNumber: a.cfg.Breaker.RepeatNumber,
	}
	openWeather := serviceWeather.NewBreakerClient("OpenWeather", breakerCfg,
		serviceWeather.NewClientOpenWeatherMap(a.cfg.OpenWeatherMapAPIKey, a.cfg.OpenWeatherMapURL, httpLogClient, a.l),
	)
	weatherAPI := serviceWeather.NewBreakerClient("WeatherAPI", breakerCfg,
		serviceWeather.NewClientWeatherAPI(a.cfg.WeatherAPIKey, a.cfg.WeatherAPIURL, httpLogClient, a.l),
	)

	weatherBit := serviceWeather.NewBreakerClient("WeatherBit", breakerCfg,
		serviceWeather.NewClientWeatherBit(
			a.cfg.WeatherBitAPIKey,
			a.cfg.WeatherBitURL,
			httpLogClient,
			a.l,
		),
	)
	rawService := serviceWeather.NewService(a.l, weatherAPI, openWeather, weatherBit)

	// Metrics for cache and service
	cacheMetrics := cache.NewMetricsDecorator[models.WeatherData](
		cache.NewRedisClient[models.WeatherData](redisClient, a.l, time.Duration(a.cfg.Redis.LiveTime)*time.Hour),
		metricsSvc.NewPromCollector(),
	)
	weatherService := decorators.NewCachedService(rawService, cacheMetrics, a.l)

	// Setup Gin router
	router := gin.New()
	router.Use(gin.Recovery())

	// Setup gRPC server with metrics interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(a.m.UnaryInterceptor()),
		grpc.StreamInterceptor(a.m.StreamInterceptor()),
	)
	weather.RegisterWeatherServiceServer(grpcServer, grpc2.NewWeatherGRPCServer(weatherService))

	// HTTP server config (unused but prepared)
	httpServer := &http.Server{
		Addr:        a.cfg.ServerAddress(),
		Handler:     router,
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout) * time.Second,
	}

	// Start gRPC server
	addrGrpc := a.cfg.Server.Host + ":" + a.cfg.Server.GrpcPort
	go func() {
		a.l.Info().Str("address", addrGrpc).Msg("gRPC server running")
		l, lErr := net.Listen("tcp", addrGrpc)
		if lErr != nil {
			a.l.Error().Err(lErr).Msg("failed to listen on gRPC port")
			return
		}
		if serveErr := grpcServer.Serve(l); serveErr != nil {
			a.l.Error().Err(serveErr).Msg("gRPC server failed")
		}
	}()

	srvContainer := ServiceContainer{
		WeatherService: weatherService,
		GrpcServer:     grpcServer,
		Router:         router,
		Srv:            httpServer,
		fileLogger:     fileLogger,
	}

	return srvContainer
}

func newRedisConnection(connString string, dbType int) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: connString, DB: dbType})
}
