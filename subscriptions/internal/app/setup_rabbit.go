package app

import (
	"github.com/Nazarious-ucu/weather-subscription-api/pkg/messaging"
	"github.com/wagslane/go-rabbitmq"
)

func (a *App) setupConn() (*rabbitmq.Conn, error) {
	conn, err := rabbitmq.NewConn(
		a.cfg.RabbitMQ.Address(),
		rabbitmq.WithConnectionOptionsLogging,
	)
	if err != nil {
		a.l.Printf("Failed to connect to RabbitMQ: %v", err)
		return nil, err
	}

	a.l.Println("Connected to RabbitMQ successfully")
	return conn, nil
}

// Create a new publisher for subscription events
func (a *App) setupPublisher(conn *rabbitmq.Conn) (*rabbitmq.Publisher, error) {
	publisher, err := rabbitmq.NewPublisher(
		conn,
		rabbitmq.WithPublisherOptionsExchangeName(messaging.ExchangeName),
		rabbitmq.WithPublisherOptionsExchangeDeclare,
		rabbitmq.WithPublisherOptionsLogging,
		rabbitmq.WithPublisherOptionsExchangeDurable,
	)
	if err != nil {
		return nil, err
	}

	publisher.NotifyReturn(func(r rabbitmq.Return) {
		a.l.Printf("message returned from server: %s", string(r.Body))
		if r.ReplyCode != 0 {
			a.l.Printf("Message returned with reply code %d: %s", r.ReplyCode, r.RoutingKey)
		}
	})

	return publisher, nil
}
