package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/cfg"
	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/handlers/subscription"
	weatherHTTP "github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/handlers/weather"
	"github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/subs"
	weatherpb "github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/weather"
)

type App struct {
	cfg cfg.Config
	log *zap.SugaredLogger
}

func New(cfg cfg.Config, logger *zap.SugaredLogger) *App {
	return &App{
		cfg: cfg,
		log: logger,
	}
}

func (a *App) Start(ctx context.Context) error {
	mux := runtime.NewServeMux()
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	a.log.Infow("registering SubscriptionService handler",
		"endpoint", a.cfg.SubServer.Address(),
	)
	if err := subs.RegisterSubscriptionServiceHandlerFromEndpoint(
		ctx,
		mux,
		a.cfg.SubServer.Address(),
		dialOpts); err != nil {
		a.log.Errorw("failed to register SubscriptionService handler", "error", err)
		return err
	}

	a.log.Infow("registering WeatherService handler",
		"endpoint", a.cfg.WeatherServer.Address(),
	)
	if err := weatherpb.RegisterWeatherServiceHandlerFromEndpoint(
		ctx,
		mux,
		a.cfg.WeatherServer.Address(),
		dialOpts); err != nil {
		a.log.Errorw("failed to register WeatherService handler", "error", err)
		return err
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://"+a.cfg.ServerAddress()+"/swagger/swagger.json"),
	))

	httpMux.Handle("/", mux)

	apiServer := &http.Server{
		Addr:        a.cfg.ServerAddress(),
		Handler:     httpMux,
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout) * time.Second,
	}

	// Handlers using your HTTP client + Zap
	subHandler := subscription.NewHandler(
		&http.Client{},
		a.cfg.SubServer.Host+":"+a.cfg.SubServer.HTTPPort,
		a.log,
	)
	weathHandler := weatherHTTP.NewHandler(
		&http.Client{},
		a.cfg.WeatherServer.Host+":"+a.cfg.WeatherServer.HTTPPort,
		a.log,
	)
	subHandler.RegisterRoutes(httpMux)
	weathHandler.RegisterRoutes(httpMux)

	// Launch server
	go func() {
		a.log.Infow("starting gateway HTTP server", "address", a.cfg.ServerAddress())
		if err := apiServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.log.Errorw("gateway server error", "error", err)
		}
	}()

	<-ctx.Done()
	a.log.Infow("shutdown signal received, stopping gateway")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(a.cfg.Server.ReadTimeout)*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		a.log.Errorw("forced shutdown due to error", "error", err)
	} else {
		a.log.Infow("gateway exited gracefully")
	}

	return nil
}
