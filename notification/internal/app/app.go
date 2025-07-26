package app

import (
	"context"
	"log"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/consumer"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/emailer"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/services/email"
)

const (
	timeoutDuration = 5 * time.Second
)

type ServiceContainer struct {
	EmailService *email.Service
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
	a.log.Println("Starting application...")
	// ctxWithTimeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	// defer cancel()

	smtpService := emailer.NewSMTPService(&a.cfg, a.log)
	a.log.Printf("Initializing SMTP service with config: %+v\n", a.cfg.Email)
	emailService := email.NewService(smtpService, a.cfg.TemplatesDir)

	rabbitConn, err := a.setupConn()
	if err != nil {
		a.log.Fatalf("Failed to connect to RabbitMQ: %v", err)
		return err
	}

	weatherEventConsumer, err := a.setupWeatherConsumer(rabbitConn)
	if err != nil {
		a.log.Fatalf("Failed to setup weather consumer: %v", err)
		return err
	}
	defer weatherEventConsumer.Close()

	subscribeEventConsumer, err := a.setupSubscribeEventConsumer(rabbitConn)
	if err != nil {
		a.log.Fatalf("Failed to setup subscribe event consumer: %v", err)
		return err
	}
	defer subscribeEventConsumer.Close()

	customConsumer := consumer.NewConsumer(emailService, a.log)

	err = weatherEventConsumer.Run(customConsumer.ReceiveWeather)
	if err != nil {
		a.log.Fatalf("Failed to run weather event consumer: %v", err)
		return err
	}

	err = subscribeEventConsumer.Run(customConsumer.ReceiveSubscription)
	if err != nil {
		a.log.Fatalf("Failed to run subscribe event consumer: %v", err)
	}

	a.log.Println("Application started successfully ")

	<-ctx.Done()

	// if err := a.Stop(srvContainer); err != nil {
	//	a.log.Fatalf("failed to shutdown application: %v", err)
	//	return err
	// }
	a.log.Println("Application shutdown successfully")
	return nil
}
