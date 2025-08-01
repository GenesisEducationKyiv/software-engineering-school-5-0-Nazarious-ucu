package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/metrics"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog"
	httpSwagger "github.com/swaggo/http-swagger"
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
	l   zerolog.Logger
}

func New(cfg cfg.Config, logger zerolog.Logger) *App {
	return &App{
		cfg: cfg,
		l:   logger,
	}
}

func (a *App) Start(ctx context.Context) error {
	m := metrics.NewMetrics("gateway")

	mux := runtime.NewServeMux()
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	a.l.Info().
		Str("endpoint", a.cfg.SubServer.Address()).
		Msg("registering SubscriptionService handler")
	if err := subs.RegisterSubscriptionServiceHandlerFromEndpoint(
		ctx,
		mux,
		a.cfg.SubServer.Address(),
		dialOpts); err != nil {
		a.l.Error().
			Err(err).
			Msg("failed to register SubscriptionService handler")
		return err
	}

	a.l.Info().
		Str("endpoint", a.cfg.WeatherServer.Address()).
		Msg("registering WeatherService handler")
	if err := weatherpb.RegisterWeatherServiceHandlerFromEndpoint(
		ctx,
		mux,
		a.cfg.WeatherServer.Address(),
		dialOpts); err != nil {
		a.l.Error().
			Err(err).
			Msg("failed to register WeatherService handler")
		return err
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://"+a.cfg.ServerAddress()+"/swagger/swagger.json"),
	))

	httpMux.Handle("/metrics", promhttp.Handler())

	apiServer := &http.Server{
		Addr:        a.cfg.ServerAddress(),
		Handler:     httpMux,
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout) * time.Second,
	}

	// Handlers using your HTTP client + Zap
	subHandler := subscription.NewHandler(
		&http.Client{},
		a.cfg.SubServer.Host+":"+a.cfg.SubServer.HTTPPort,
		a.l,
		m,
	)
	weathHandler := weatherHTTP.NewHandler(
		&http.Client{},
		a.cfg.WeatherServer.Host+":"+a.cfg.WeatherServer.HTTPPort,
		a.l,
		m,
	)

	httpMux.Handle("/api/v1/http/subscriptions/subscribe", m.InstrumentHandler(
		http.HandlerFunc(subHandler.HandleSubscribe)))
	httpMux.Handle("/api/v1/http/subscriptions/confirm/", m.InstrumentHandler(
		http.HandlerFunc(subHandler.HandleConfirm)))
	httpMux.Handle("/api/v1/http/subscriptions/unsubscribe/", m.InstrumentHandler(
		http.HandlerFunc(subHandler.HandleUnsubscribe)))

	httpMux.Handle("/api/v1/http/weather", m.InstrumentHandler(
		http.HandlerFunc(weathHandler.HandleGetWeather)))

	httpMux.Handle("/v2/", mux)
	// Launch server
	go func() {
		a.l.Info().
			Str("address", a.cfg.ServerAddress()).
			Msg("starting gateway HTTP server")
		if err := apiServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.l.Error().
				Err(err).
				Msg("gateway server error")
		}
	}()

	<-ctx.Done()
	a.l.Info().
		Msg("shutdown signal received, stopping gateway")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(a.cfg.Server.ReadTimeout)*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		a.l.Error().
			Err(err).
			Msg("forced shutdown due to error")
	} else {
		a.l.Info().
			Msg("gateway exited gracefully")
	}

	return nil
}
