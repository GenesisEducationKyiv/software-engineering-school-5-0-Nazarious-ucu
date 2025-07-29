//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/pkg/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wagslane/go-rabbitmq"

	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/app"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/config"

	"github.com/stretchr/testify/assert"
)

var (
	testServerURL string
	db            *sql.DB
	rmqConn       *rabbitmq.Conn
)

func TestMain(m *testing.M) {
	fmt.Println("Starting integration tests...")

	// Initialize the application
	cfg, err := config.NewConfig()
	if err != nil {
		log.Panicf("failed to load configuration: %v", err)
	}

	cfg.DB.Source = "test.db"
	cfg.DB.MigrationsPath = "../../migrations"

	cfg.Server.Host = "127.0.0.1"
	cfg.Server.GrpcPort = "8081"

	application := app.New(*cfg, log.Default())
	ctx := context.Background()

	database, err := app.CreateSqliteDb(ctx, cfg.DB.Dialect, cfg.DB.Source)
	if err != nil {
		log.Panicf("failed to create database: %v", err)
	}

	err = app.InitSqliteDb(database, cfg.DB.Dialect, cfg.DB.MigrationsPath)
	if err != nil {
		log.Panicf("failed to init database: %v", err)
	}

	// Check if the database is initialized using testify assert
	if database == nil {
		log.Panic("Database is not initialized")
	}

	// Check if is there new table in the database
	if err := database.Ping(); err != nil {
		log.Panicf("failed to connect to the database: %v", err)
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)

	go func() {
		if err := application.Start(ctxWithCancel); err != nil {
			log.Panic(err)
		}
	}()

	rmqConn, err = rabbitmq.NewConn(
		cfg.RabbitMQ.Address(),
	)
	if err != nil {
		log.Panicf("failed to connect to RabbitMQ: %v", err)
	}
	if err := forceDeclareRabbitQueue(cfg.RabbitMQ.Address()); err != nil {
		log.Panicf("failed to declare queue and binding manually: %v", err)
	}

	initIntegration("http://"+cfg.ServerAddress(), database)
	time.Sleep(100 * time.Millisecond)

	// Run the tests
	_ = m.Run()

	cancel()
}

func resetTables(db *sql.DB) error {
	// Reset the database tables before each test
	_, err := db.Exec("DELETE FROM subscriptions")
	if err != nil {
		return fmt.Errorf("failed to reset subscriptions table: %w", err)
	}
	return nil
}

func initIntegration(serverURL string, database *sql.DB) {
	testServerURL = serverURL
	db = database
}

func FetchSubscription(t *testing.T, email, city string) map[string]interface{} {
	row := db.QueryRow(
		`SELECT email, city, frequency FROM subscriptions WHERE email = ? AND city = ?`,
		email, city,
	)
	var e, c, freq string
	err := row.Scan(&e, &c, &freq)

	assert.NoErrorf(t, err, "failed to fetch subscription: %v", err)

	count := db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE email = ? AND city = ?`, email, city)

	var cnt int
	err = count.Scan(&cnt)
	assert.NoErrorf(t, err, "failed to count subscriptions: %v", err)

	return map[string]interface{}{
		"email":     e,
		"city":      c,
		"frequency": freq,
		"Count":     cnt,
	}
}

func saveSubscription(t *testing.T, email, city string, freq string, token string) {
	_, err := db.Exec(
		`INSERT INTO subscriptions (email, city, frequency, token) VALUES (?, ?, ?, ?)`,
		email, city, freq, token,
	)
	assert.NoErrorf(t, err, "failed to save subscription: %v", err)
}

func readLatestRabbitMQMessage(consumer *rabbitmq.Consumer, queue string) ([]byte, error) {
	var body []byte
	read := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		err := consumer.Run(func(d rabbitmq.Delivery) rabbitmq.Action {
			body = d.Body
			close(read)
			return rabbitmq.Ack
		})
		if err != nil {
			log.Printf("Consumer run error: %v", err)
		}
	}()

	select {
	case <-read:
		return body, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for message from queue %s", queue)
	}
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

	log.Println("Queue and binding ensured manually via amqp091")
	return nil
}
