//go:build integration

package app

import (
	"context"
	"fmt"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/metrics"
	"github.com/Nazarious-ucu/weather-subscription-api/pkg/logger"
	"log"
	"testing"
	"time"

	"github.com/wagslane/go-rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/pkg/messaging"
)

var rmqConn *rabbitmq.Conn

func TestMain(m *testing.M) {
	fmt.Println("Starting integration tests...")

	// Initialize the application
	cfg, err := config.NewConfig()
	if err != nil {
		log.Panicf("failed to load configuration: %v", err)
	}

	cfg.Email.Host = "localhost"
	cfg.Email.Port = "1025"

	l, err := logger.NewLogger("log/notifications-test.log", "notification_service_test")
	if err != nil {
		log.Panicf("failed to initialize logger: %v", err)
	}

	met := metrics.NewMetrics("notification_service_test")

	application := app.New(*cfg, l, met)
	ctx := context.Background()

	ctxWithCancel, cancel := context.WithCancel(ctx)

	if err := forceDeclareRabbitQueue(cfg.RabbitMQ.Address()); err != nil {
		log.Panicf("failed to declare queue and binding manually: %v", err)
	}

	rmqConn, err = rabbitmq.NewConn(
		cfg.RabbitMQ.Address(),
	)
	if err != nil {
		log.Panicf("failed to connect to RabbitMQ: %v", err)
	}
	go func() {
		if err := application.Start(ctxWithCancel); err != nil {
			log.Panic(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Run the tests
	_ = m.Run()

	cancel()
	// os.Exit(code)
}

func forceDeclareRabbitQueue(amqpURL string) error {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return fmt.Errorf("amqp dial failed: %w", err)
	}
	defer func(conn *amqp.Connection) {
		err := conn.Close()
		if err != nil {
			log.Printf("Failed to close AMQP connection: %v", err)
		} else {
			log.Println("AMQP connection closed successfully")
		}
	}(conn)

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("amqp channel error: %w", err)
	}
	defer func(ch *amqp.Channel) {
		err := ch.Close()
		if err != nil {
			log.Printf("Failed to close AMQP channel: %v", err)
		} else {
			log.Println("AMQP channel closed successfully")
		}
	}(ch)

	if err := ch.ExchangeDeclare(
		messaging.ExchangeName,
		"direct",
		true, // durable
		false, false, false,
		nil,
	); err != nil {
		return fmt.Errorf("exchange declare error: %w", err)
	}

	_, err = ch.QueueDeclare(
		messaging.SubscribeQueueName,
		true, false, false, false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("queue declare error: %w", err)
	}

	if err := ch.QueueBind(
		messaging.SubscribeQueueName,
		messaging.SubscribeRoutingKey,
		messaging.ExchangeName,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("queue bind error: %w", err)
	}

	_, err = ch.QueueDeclare(
		messaging.WeatherQueueName,
		true, false, false, false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("queue declare error: %w", err)
	}

	if err := ch.QueueBind(
		messaging.WeatherQueueName,
		messaging.WeatherRoutingKey,
		messaging.ExchangeName,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("queue bind error: %w", err)
	}

	log.Println("Queue and binding ensured manually via amqp091")
	return nil
}
