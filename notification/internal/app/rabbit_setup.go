package app

import (
	"github.com/Nazarious-ucu/weather-subscription-api/pkg/messaging"
	"github.com/wagslane/go-rabbitmq"
)

func (a *App) setupConn() (*rabbitmq.Conn, error) {
	conn, err := rabbitmq.NewConn(
		a.cfg.RabbitMQ.Address(),
	)
	if err != nil {
		a.log.Fatalf("Failed to connect to RabbitMQ: %v", err)
		return nil, err
	}

	a.log.Println("Connected to RabbitMQ successfully")
	return conn, nil
}

// Create a new consumer for subscription events
func (a *App) setupSubscribeEventConsumer(conn *rabbitmq.Conn) (*rabbitmq.Consumer, error) {
	consumer, err := rabbitmq.NewConsumer(
		conn,
		messaging.SubscribeQueueName,
		rabbitmq.WithConsumerOptionsExchangeName(messaging.ExchangeName),
		rabbitmq.WithConsumerOptionsExchangeDeclare,
		rabbitmq.WithConsumerOptionsExchangeDurable,
		rabbitmq.WithConsumerOptionsRoutingKey(messaging.SubscribeRoutingKey),
		rabbitmq.WithConsumerOptionsQueueDurable,
	)
	if err != nil {
		return nil, err
	}
	return consumer, nil
}

// Create a new consumer for weather commands
func (a *App) setupWeatherConsumer(conn *rabbitmq.Conn) (*rabbitmq.Consumer, error) {
	consumer, err := rabbitmq.NewConsumer(
		conn,
		messaging.WeatherQueueName,
		rabbitmq.WithConsumerOptionsExchangeName(messaging.ExchangeName),
		rabbitmq.WithConsumerOptionsExchangeDeclare,
		rabbitmq.WithConsumerOptionsExchangeDurable,
		rabbitmq.WithConsumerOptionsRoutingKey(messaging.WeatherRoutingKey),
		rabbitmq.WithConsumerOptionsQueueDurable,
	)
	if err != nil {
		return nil, err
	}
	return consumer, nil
}
