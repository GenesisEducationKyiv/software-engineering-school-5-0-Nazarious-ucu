package app

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	grpc2 "github.com/Nazarious-ucu/weather-subscription-api/weather/internal/handlers/grpc"
	http2 "github.com/Nazarious-ucu/weather-subscription-api/weather/internal/handlers/http"
	"github.com/gin-gonic/gin"

	weatherpb "github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/weather"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/cache"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/logger"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/metrics"
	serviceWeather "github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/weather"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/weather/decorators"
	fLogger "github.com/Nazarious-ucu/weather-subscription-api/weather/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type ServiceContainer struct {
	WeatherService *decorators.CachedService
	GrpcServer     *grpc.Server

	Router     *gin.Engine
	Srv        *http.Server
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
	srvContainer := a.init(ctx)

	a.log.Println("starting weather service on:", a.cfg.Server.GrpcPort)

	router := http.NewServeMux()

	router.Handle("/metrics", promhttp.Handler())

	weatherHandler := http2.NewHandler(srvContainer.WeatherService)

	srvContainer.Router.GET("/weather", weatherHandler.GetWeather)

	a.log.Println("Weather service started successfully on", a.cfg.Server.GrpcPort)

	<-ctx.Done()

	if err := a.Shutdown(srvContainer); err != nil {
		a.log.Printf("failed to shutdown application: %v", err)
		return err
	}
	a.log.Println("Application shutdown successfully")

	return nil
}

func (a *App) Shutdown(srvContainer ServiceContainer) error {
	a.log.Println("Stopping weather service…")

	defer func(fileLogger *zap.Logger) {
		err := fileLogger.Sync()
		if err != nil {
			a.log.Printf("failed to sync file logger: %v", err)
		} else {
			a.log.Println("File logger synced successfully")
		}
	}(srvContainer.fileLogger)

	log.Println("Shutting down gRPC server...")
	srvContainer.GrpcServer.GracefulStop()

	a.log.Println("Shutdown complete")
	return nil
}

func (a *App) init(ctx context.Context) ServiceContainer {
	a.log.Println("Initializing weather service with configuration:", a.cfg)

	redisClient := newRedisConnection(a.cfg.Redis.Host+":"+a.cfg.Redis.Port, a.cfg.Redis.DbType)

	fileLogger, err := fLogger.NewFileLogger(a.cfg.LogsPath)
	if err != nil {
		a.log.Printf("failed to create file logger: %v", err)
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

	addrGrpc := a.cfg.Server.Host + ":" + a.cfg.Server.GrpcPort

	router := gin.Default()

	apiServer := &http.Server{
		Addr:        a.cfg.ServerAddress(),
		Handler:     router,
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout) * time.Second,
	}

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", addrGrpc)
	if err != nil {
		a.log.Println(err)
	}
	grpcServer := grpc.NewServer()

	weatherpb.RegisterWeatherServiceServer(grpcServer, grpc2.NewWeatherGRPCServer(cacheDecorator))

	a.log.Printf("Application initialized on address: %s", apiServer.Addr)

	go func() {
		log.Printf("gRPC server running at %s", addrGrpc)
		if err := grpcServer.Serve(lis); err != nil {
			a.log.Printf("gRPC server failed: %v", err)
		}
	}()

	srvContainer := ServiceContainer{
		WeatherService: cacheDecorator,
		GrpcServer:     grpcServer,

		Router:     router,
		Srv:        apiServer,
		fileLogger: fileLogger,
	}

	return srvContainer
}

func newRedisConnection(connString string, dbType int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: connString,
		DB:   dbType,
	})
}
