package producers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/pkg/messaging"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/metrics"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/models"
	"github.com/rs/zerolog"
	"github.com/wagslane/go-rabbitmq"
)

// Producer publishes events to RabbitMQ with structured logging and metrics.
type Producer struct {
	prod *rabbitmq.Publisher
	log  zerolog.Logger
	m    *metrics.Metrics
}

// NewProducer initializes a Producer with logger context and metrics collector.
func NewProducer(
	prod *rabbitmq.Publisher,
	logger zerolog.Logger,
	m *metrics.Metrics,
) *Producer {
	// enrich logger with component
	logger = logger.With().Str("component", "RabbitMQProducer").Logger()
	return &Producer{prod: prod, log: logger, m: m}
}

// Publish sends a raw message to the given routing keys, recording logs and metrics.
func (p *Producer) Publish(
	ctx context.Context,
	routingKey []string,
	body []byte,
) error {
	start := time.Now()
	p.log.Debug().Ctx(ctx).
		Strs("routing_key", routingKey).
		Msg("publishing message to exchange")

	err := p.prod.PublishWithContext(
		ctx,
		body,
		routingKey,
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsMandatory,
		rabbitmq.WithPublishOptionsPersistentDelivery,
		rabbitmq.WithPublishOptionsExchange(messaging.ExchangeName),
	)
	dur := time.Since(start)

	if err != nil {
		p.log.Error().
			Err(err).Ctx(ctx).
			Strs("routing_key", routingKey).
			Dur("duration", dur).
			Msg("failed to publish message")

		// record a publishing failure metric, if desired
		p.m.TechnicalErrors.WithLabelValues("rabbitmq_publish", "critical").Inc()
		return err
	}

	p.log.Info().Ctx(ctx).
		Strs("routing_key", routingKey).
		Dur("duration", dur).
		Msg("message published successfully")

	p.m.RabbitPublishTotal.WithLabelValues("rabbitmq_publish", "success").Inc()

	return nil
}

// SendWeather marshals a WeatherNotifyEvent and publishes it.
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
		p.log.Error().
			Err(err).Ctx(ctx).
			Str("email", email).
			Msg("failed to marshal WeatherNotifyEvent")
		return err
	}

	return p.Publish(ctx, []string{messaging.WeatherRoutingKey}, body)
}

// SendConfirmation marshals a NewSubscriptionEvent and publishes it.
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
		p.log.Error().
			Err(err).Ctx(ctx).
			Str("email", email).
			Msg("failed to marshal NewSubscriptionEvent")
		return err
	}

	return p.Publish(ctx, []string{messaging.SubscribeRoutingKey}, body)
}
