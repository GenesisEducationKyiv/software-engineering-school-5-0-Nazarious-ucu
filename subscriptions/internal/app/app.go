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

	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/emailer"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/handlers/subscription"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/notifier"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/repository/sqlite"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/services/email"
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
	EmailService        *email.Service
	Notificator         *notifier.Notifier
	SubRepository       sqlite.SubscriptionRepository
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
	a.log.Println("Starting server on", a.cfg.Server.Port)

	defer func() {
		if err := srvContainer.Srv.Close(); err != nil {
			a.log.Println("Error stopping server:", err)
		}
	}()

	subHandler := subscription.NewHandler(srvContainer.SubscriptionService)

	api := srvContainer.Router.Group("/api")
	{
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

	a.log.Println("Application started successfully on", a.cfg.Server.Port)
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

	router := gin.Default()

	apiServer := &http.Server{
		Addr:        a.cfg.ServerAddress(),
		Handler:     router,
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout) * time.Second,
	}

	a.log.Printf("Application initialized on address: %s", apiServer.Addr)

	smtpService := emailer.NewSMTPService(&a.cfg, a.log)
	a.log.Printf("Initializing SMTP service with config: %+v\n", a.cfg.Email)
	subRepository := sqlite.NewSubscriptionRepository(db, a.log)
	emailService := email.NewService(smtpService, a.cfg.TemplatesDir)

	subService := subs2.NewService(subRepository, emailService)

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx,
		"tcp",
		a.cfg.Server.Address+":50051")
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

	weatherAdapter := weather.NewGrpcWeatherAdapter(weatherGrpc, a.log)

	subs.RegisterSubscriptionServiceServer(grpcServer, subs2.NewSubscriptionGRPCServer(subService))

	go func() {
		log.Println("gRPC server running at :50051")
		if err := grpcServer.Serve(lis); err != nil {
			a.log.Panicf("gRPC server failed: %v", err)
		}
	}()

	notificator := notifier.New(subRepository,
		weatherAdapter,
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
