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

	//defer consumer.Close()
	//
	//err = consumer.Run(func(d rabbitmq.Delivery) rabbitmq.Action {
	//	logger.Printf("📧 Sending confirmation email: %s", string(d.Body))
	//	return rabbitmq.Ack
	//})
	//if err != nil {
	//	logger.Fatalf("consumer failed to run: %v", err)
	//}
}

// Create a new consumer for subscription events
func (a *App) setupSubscribeEventConsumer(conn *rabbitmq.Conn) (*rabbitmq.Consumer, error) {
	consumer, err := rabbitmq.NewConsumer(
		conn,
		messaging.SubscribeQueueName,
		rabbitmq.WithConsumerOptionsExchangeName("notifications"),
		rabbitmq.WithConsumerOptionsExchangeDeclare,
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
		rabbitmq.WithConsumerOptionsExchangeName("notifications"),
		rabbitmq.WithConsumerOptionsExchangeDeclare,
		rabbitmq.WithConsumerOptionsRoutingKey(messaging.WeatherDailyRoutingKey),
		rabbitmq.WithConsumerOptionsRoutingKey(messaging.WeatherHourlyRoutingKey),
		rabbitmq.WithConsumerOptionsQueueDurable,
	)
	if err != nil {
		return nil, err
	}
	return consumer, nil
}
