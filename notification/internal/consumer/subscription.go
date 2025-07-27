package consumer

import (
	"encoding/json"
	"log"

	"github.com/Nazarious-ucu/weather-subscription-api/pkg/messaging"
	"github.com/wagslane/go-rabbitmq"

	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/models"
)

type emailSender interface {
	SendConfirmation(email, token string) error
	SendWeather(to string, forecast models.WeatherData) error
}

type Consumer struct {
	emailSender emailSender
	logger      *log.Logger
}

func NewConsumer(emailSender emailSender, logger *log.Logger) *Consumer {
	return &Consumer{
		emailSender: emailSender,
		logger:      logger,
	}
}

func (c *Consumer) ReceiveSubscription(d rabbitmq.Delivery) rabbitmq.Action {
	c.logger.Printf("Sending subscription confirmation: %s", string(d.Body))

	var event messaging.NewSubscriptionEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return rabbitmq.NackDiscard
	}

	if err := c.emailSender.SendConfirmation(event.Email, event.Token); err != nil {
		c.logger.Printf("Error sending confirmation email: %v", err)
		return rabbitmq.NackDiscard
	}

	return rabbitmq.Ack
}

func (c *Consumer) ReceiveWeather(d rabbitmq.Delivery) rabbitmq.Action {
	c.logger.Printf("Sending weather data: %s", string(d.Body))
	var weatherData models.WeatherData
	var event messaging.WeatherNotifyEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return rabbitmq.NackDiscard
	}

	if err := c.emailSender.SendWeather(event.Email, weatherData); err != nil {
		c.logger.Printf("Error sending weather email: %v", err)
		return rabbitmq.NackDiscard
	}

	return rabbitmq.Ack
}
