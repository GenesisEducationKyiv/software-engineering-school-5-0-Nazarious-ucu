package app

import (
	"context"
	"log"

	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/consumer"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/emailer"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/services/email"
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
		a.log.Printf("Failed to connect to RabbitMQ: %v", err)
		return err
	}

	subscribeEventConsumer, err := a.setupSubscribeEventConsumer(rabbitConn)
	if err != nil {
		a.log.Printf("Failed to setup subscribe event consumer: %v", err)
		return err
	}
	defer subscribeEventConsumer.Close()

	weatherEventConsumer, err := a.setupWeatherConsumer(rabbitConn)
	if err != nil {
		a.log.Printf("Failed to setup weather consumer: %v", err)
		return err
	}
	defer weatherEventConsumer.Close()

	customConsumer := consumer.NewConsumer(emailService, a.log)
	go func() {
		// defer wg.Done()
		if err := weatherEventConsumer.Run(customConsumer.ReceiveWeather); err != nil {
			a.log.Printf("weather consumer stopped: %v", err)
		}
	}()

	// 4) Launch subscribe consumer
	go func() {
		// defer wg.Done()
		if err := subscribeEventConsumer.Run(customConsumer.ReceiveSubscription); err != nil {
			a.log.Printf("subscribe consumer stopped: %v", err)
		}
	}()
	a.log.Println("Application started successfully ")

	<-ctx.Done()

	// if err := a.Stop(srvContainer); err != nil {
	//	a.log.Fatalf("failed to shutdown application: %v", err)
	//	return err
	// }
	a.log.Println("Application shutdown successfully")
	return nil
}
