package consumer

import (
	"encoding/json"

	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/metrics"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/pkg/messaging"
	"github.com/rs/zerolog"
	"github.com/wagslane/go-rabbitmq"
)

type emailSender interface {
	SendConfirmation(email, token string) error
	SendWeather(to string, forecast models.WeatherData) error
}

// Consumer processes RabbitMQ deliveries and emits logs & metrics.
type Consumer struct {
	emailSender emailSender
	logger      zerolog.Logger
	m           *metrics.Metrics
}

// NewConsumer wires up the email sender, a scoped logger, and the metrics collector.
func NewConsumer(emailSender emailSender, logger zerolog.Logger, m *metrics.Metrics) *Consumer {
	logger = logger.With().Str("component", "Consumer").Logger()
	return &Consumer{
		emailSender: emailSender,
		logger:      logger,
		m:           m,
	}
}

// ReceiveSubscription handles NewSubscriptionEvent messages.
func (c *Consumer) ReceiveSubscription(d rabbitmq.Delivery) rabbitmq.Action {
	const eventType = messaging.SubscribeRoutingKey

	// record that we got a message
	c.m.ConsumerMessagesTotal.WithLabelValues(eventType).Inc()
	c.logger.Debug().
		Str("payload", string(d.Body)).
		Msg("received subscription event")

	// parse
	var evt messaging.NewSubscriptionEvent
	if err := json.Unmarshal(d.Body, &evt); err != nil {
		c.logger.Error().
			Err(err).
			Str("event", eventType).
			Msg("unmarshal error")
		c.m.ConsumerErrorsTotal.WithLabelValues(eventType, "unmarshal_error").Inc()
		c.m.ConsumerMessagesTotal.WithLabelValues(eventType, "error").Inc()
		return rabbitmq.NackDiscard
	}

	// send confirmation email
	c.m.EmailSentTotal.WithLabelValues(eventType).Inc()
	if err := c.emailSender.SendConfirmation(evt.Email, evt.Token); err != nil {
		c.logger.Error().
			Err(err).
			Str("email", evt.Email).
			Msg("failed to send confirmation email")
		c.m.ConsumerErrorsTotal.WithLabelValues(eventType, "send_email_error").Inc()
		c.m.ConsumerMessagesTotal.WithLabelValues(eventType, "error").Inc()
		return rabbitmq.NackDiscard
	}

	c.logger.Info().
		Str("email", evt.Email).
		Msg("confirmation email sent")
	c.m.ConsumerMessagesTotal.WithLabelValues(eventType).Inc()
	return rabbitmq.Ack
}

// ReceiveWeather handles WeatherNotifyEvent messages.
func (c *Consumer) ReceiveWeather(d rabbitmq.Delivery) rabbitmq.Action {
	const eventType = "weather"

	c.m.ConsumerMessagesTotal.WithLabelValues(eventType).Inc()
	c.logger.Debug().
		Str("payload", string(d.Body)).
		Msg("received weather event")

	var evt messaging.WeatherNotifyEvent
	if err := json.Unmarshal(d.Body, &evt); err != nil {
		c.logger.Error().
			Err(err).
			Str("event", eventType).
			Msg("unmarshal error")
		c.m.ConsumerErrorsTotal.WithLabelValues(eventType, "unmarshal_error").Inc()
		c.m.ConsumerMessagesTotal.WithLabelValues(eventType).Inc()
		return rabbitmq.NackDiscard
	}

	fc := models.WeatherData{
		City:        evt.Weather.City,
		Temperature: evt.Weather.Temperature,
		Condition:   evt.Weather.Description,
	}

	c.m.EmailSentTotal.WithLabelValues(eventType).Inc()
	if err := c.emailSender.SendWeather(evt.Email, fc); err != nil {
		c.logger.Error().
			Err(err).
			Str("email", evt.Email).
			Msg("failed to send weather email")
		c.m.EmailErrorsTotal.WithLabelValues(eventType).Inc()
		c.m.ConsumerErrorsTotal.WithLabelValues(eventType, "send_email_error").Inc()
		c.m.ConsumerMessagesTotal.WithLabelValues(eventType).Inc()
		return rabbitmq.NackDiscard
	}

	c.logger.Info().
		Str("email", evt.Email).
		Str("city", fc.City).
		Msg("weather email sent")
	c.m.ConsumerMessagesTotal.WithLabelValues(eventType).Inc()
	return rabbitmq.Ack
}
