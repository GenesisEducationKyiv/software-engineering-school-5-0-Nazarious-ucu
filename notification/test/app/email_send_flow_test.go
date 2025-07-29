//go:build integration

package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wagslane/go-rabbitmq"
)

func TestConfirmationEmailSent(t *testing.T) {
	// Надіслати подію вручну в RabbitMQ
	event := messaging.NewSubscriptionEvent{
		Email: "test@gmail.com",
		Token: "abcd-1234",
	}
	body, err := json.Marshal(event)
	require.NoError(t, err)

	pub, err := rabbitmq.NewPublisher(rmqConn,
		rabbitmq.WithPublisherOptionsExchangeName(messaging.ExchangeName),
		rabbitmq.WithPublisherOptionsExchangeDeclare,
		rabbitmq.WithPublisherOptionsExchangeDurable,
	)
	require.NoError(t, err)

	defer pub.Close()

	err = pub.Publish(
		body,
		[]string{messaging.SubscribeRoutingKey},
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsExchange(messaging.ExchangeName),
	)
	require.NoError(t, err)

	pub.NotifyReturn(func(r rabbitmq.Return) {
		log.Printf("message returned from server: %s", string(r.Body))
		if r.ReplyCode != 0 {
			log.Printf("Message returned with reply code %d: %s", r.ReplyCode, r.RoutingKey)
		}
	})

	// send request to smtp server to check if email was sent
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		"http://localhost:8025/api/v2/messages", nil)
	require.NoError(t, err, "Failed to create request to SMTP server")

	<-time.After(2000 * time.Millisecond)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Failed to get messages from SMTP server")
	defer func(body io.ReadCloser) {
		err := body.Close()
		require.NoError(t, err, "Failed to close response body")
	}(resp.Body)

	// Check was the email sent to the SMTP server
	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200 from SMTP server")
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body from SMTP server")
	bodyString := string(bodyBytes)
	assert.Contains(t, bodyString, event.Token,
		"Expected email to be sent to the SMTP server")
}

func TestWeatherEmailSent(t *testing.T) {
	event := messaging.WeatherNotifyEvent{
		Email: "test@gmail.com",
		Weather: messaging.Weather{
			Temperature: 30,
			City:        "Lviv",
			Description: "Sunny",
		},
	}

	body, err := json.Marshal(event)
	require.NoError(t, err)

	pub, err := rabbitmq.NewPublisher(rmqConn,
		rabbitmq.WithPublisherOptionsExchangeName(messaging.ExchangeName),
		rabbitmq.WithPublisherOptionsExchangeDeclare,
		rabbitmq.WithPublisherOptionsExchangeDurable,
	)
	require.NoError(t, err)
	defer pub.Close()

	err = pub.Publish(
		body,
		[]string{messaging.WeatherRoutingKey},
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsExchange(messaging.ExchangeName),
	)
	require.NoError(t, err)

	<-time.After(1000 * time.Millisecond)

	// send request to smtp server to check if email was sent
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		"http://localhost:8025/api/v2/messages", nil)
	require.NoError(t, err, "Failed to create request to SMTP server")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Failed to get messages from SMTP server")

	defer func(body io.ReadCloser) {
		err := body.Close()
		require.NoError(t, err, "Failed to close response body")
	}(resp.Body)

	// Check was the email sent to the SMTP server
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200 from SMTP server")
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body from SMTP server")
	bodyString := string(bodyBytes)
	assert.Contains(t, bodyString, event.Email,
		"Expected email to be sent to the SMTP server")
	assert.Contains(t, bodyString, event.Weather.City,
		"Expected email to contain city name")
	assert.Contains(t, bodyString, event.Weather.Description,
		"Expected email to contain weather description")
}
