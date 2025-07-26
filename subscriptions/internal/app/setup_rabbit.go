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

// Create a new publisher for subscription events
func (a *App) setupPublisher(conn *rabbitmq.Conn) (*rabbitmq.Publisher, error) {
	publisher, err := rabbitmq.NewPublisher(
		conn,
		rabbitmq.WithPublisherOptionsExchangeName(messaging.ExchangeName),
	)
	if err != nil {
		return nil, err
	}
	return publisher, nil
}
