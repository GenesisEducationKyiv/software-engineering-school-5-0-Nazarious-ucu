package producers

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/Nazarious-ucu/weather-subscription-api/pkg/messaging"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/models"
	"github.com/wagslane/go-rabbitmq"
)

type Producer struct {
	prod *rabbitmq.Publisher
	log  *log.Logger
}

func NewProducer(prod *rabbitmq.Publisher, logger *log.Logger) *Producer {
	return &Producer{
		prod: prod,
		log:  logger,
	}
}

func (p *Producer) Publish(ctx context.Context, routingKey []string, body []byte) error {
	if err := p.prod.PublishWithContext(
		ctx,
		body,
		routingKey,
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsMandatory,
	); err != nil {
		p.log.Printf("Failed to publish message: %v", err)
		return err
	}
	p.log.Printf("Message published with routing key %s", routingKey)

	p.prod.NotifyReturn(func(r rabbitmq.Return) {
		log.Printf("message returned from server: %s", string(r.Body))
	})
	return nil
}

func (p *Producer) SendWeather(
	ctx context.Context,
	email string,
	data models.WeatherData,
) error {
	event := messaging.WeatherNotifyEvent{
		Email: email,
		Weather: messaging.Weather{
			Temperature: data.Temperature,
			City:        data.City,
			Description: data.Condition,
		},
	}

	body, err := json.Marshal(event)
	if err != nil {
		p.log.Printf("Failed to marshal weather event: %v", err)
		return err
	}

	return p.Publish(ctx, strings.Split(messaging.WeatherRoutingKey, ""), body)
}

func (p *Producer) SendConfirmation(
	ctx context.Context,
	email,
	token string,
) error {
	event := messaging.NewSubscriptionEvent{
		Email: email,
		Token: token,
	}

	body, err := json.Marshal(event)
	if err != nil {
		p.log.Printf("Failed to marshal subscription event: %v", err)
		return err
	}

	return p.Publish(ctx, strings.Split(messaging.SubscribeRoutingKey, ""), body)
}
