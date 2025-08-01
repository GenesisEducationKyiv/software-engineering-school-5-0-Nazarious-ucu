package app

import (
	"context"

	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/consumer"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/emailer"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/metrics"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/services/email"
	"github.com/rs/zerolog"
)

// App is the notification application.
type App struct {
	cfg config.Config
	l   zerolog.Logger
	m   *metrics.Metrics
}

// New constructs the App with config, structured logger, and metrics.
func New(cfg config.Config, logger zerolog.Logger, m *metrics.Metrics) *App {
	logger = logger.With().
		Str("component", "App").
		Logger()
	logger.Info().Msg("App initialized")
	return &App{cfg: cfg, l: logger, m: m}
}

// Start launches both consumers and blocks until ctx is done.
func (a *App) Start(ctx context.Context) error {
	a.l.Info().Msg("Starting notification service")

	// SMTP/email setup
	smtpSvc := emailer.NewSMTPService(&a.cfg, a.l)
	a.l.Info().
		Interface("email_cfg", a.cfg.Email).
		Msg("SMTP service configured")
	emailSvc := email.NewService(smtpSvc, a.cfg.TemplatesDir)

	// RabbitMQ connection
	conn, err := a.setupConn()
	if err != nil {
		a.l.Error().Err(err).Msg("RabbitMQ connection failed")
		a.m.ConsumerErrorsTotal.WithLabelValues("connection", err.Error()).Inc()
		return err
	}

	// subscription-confirmation consumer
	subCons, err := a.setupSubscribeEventConsumer(conn)
	if err != nil {
		a.l.Error().Err(err).Msg("Subscribe consumer setup failed")
		a.m.ConsumerErrorsTotal.WithLabelValues("subscribe_setup", err.Error()).Inc()
		return err
	}
	defer subCons.Close()

	// weather-notify consumer
	weatherCons, err := a.setupWeatherConsumer(conn)
	if err != nil {
		a.l.Error().Err(err).Msg("Weather consumer setup failed")
		a.m.ConsumerErrorsTotal.WithLabelValues("weather_setup", err.Error()).Inc()
		return err
	}
	defer weatherCons.Close()

	// our processing logic
	consumerLogic := consumer.NewConsumer(emailSvc, a.l, a.m)

	// start weather consumer loop
	go func() {
		a.l.Info().Msg("Weather consumer starting")
		if err := weatherCons.Run(consumerLogic.ReceiveWeather); err != nil {
			a.l.Error().Err(err).Msg("Weather consumer stopped")
			a.m.ConsumerErrorsTotal.WithLabelValues("weather_run", err.Error()).Inc()
		}
	}()

	// start subscription-confirm consumer loop
	go func() {
		a.l.Info().Msg("Subscribe consumer starting")
		if err := subCons.Run(consumerLogic.ReceiveSubscription); err != nil {
			a.l.Error().Err(err).Msg("Subscribe consumer stopped")
			a.m.ConsumerErrorsTotal.WithLabelValues("subscribe_run", err.Error()).Inc()
		}
	}()

	a.l.Info().Msg("Notification service started")
	<-ctx.Done()
	a.l.Info().Msg("Shutdown signal received")
	return nil
}
