package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/handlers/subscription"
	weatherHTTP "github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/handlers/weather"

	httpSwagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/cfg"
	"github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/subs"
	"github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/weather"
)

type App struct {
	cfg cfg.Config
	log *log.Logger
}

func New(cfg cfg.Config, logger *log.Logger) *App {
	return &App{
		cfg: cfg,
		log: logger,
	}
}

func (a *App) Start(ctx context.Context) error {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if err := subs.RegisterSubscriptionServiceHandlerFromEndpoint(
		ctx, mux, a.cfg.SubServer.Address(), opts); err != nil {
		return err
	}
	if err := weather.RegisterWeatherServiceHandlerFromEndpoint(
		ctx, mux, a.cfg.WeatherServer.Address(), opts); err != nil {
		return err
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/swagger.json"),
	))

	httpMux.Handle("/", mux)
	apiServer := &http.Server{
		Addr:        a.cfg.ServerAddress(),
		Handler:     httpMux,
		ReadTimeout: time.Duration(a.cfg.Server.ReadTimeout) * time.Second,
	}

	subHandler := subscription.NewHandler(
		&http.Client{},
		a.cfg.SubServer.Host+":"+a.cfg.SubServer.HTTPPort,
		a.log)

	weathHandler := weatherHTTP.NewHandler(
		&http.Client{},
		a.cfg.WeatherServer.Host+":"+a.cfg.WeatherServer.HTTPPort,
		a.log)

	weathHandler.RegisterRoutes(httpMux)
	subHandler.RegisterRoutes(httpMux)

	go func() {
		a.log.Printf("Gateway listening on %s", a.cfg.ServerAddress())
		// a.log.Printf("endpoints: %s", apiServer.)apiServer
		if err := apiServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.log.Printf("server error: %v", err)
		}
	}()

	<-ctx.Done()

	a.log.Println("Shutting down gateway gracefully...")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(), time.Duration(a.cfg.Server.ReadTimeout)*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		a.log.Printf("server forced to shutdown: %v", err)
	}

	a.log.Println("Gateway exited properly")

	return nil
}
