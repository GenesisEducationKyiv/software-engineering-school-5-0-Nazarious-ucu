//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wagslane/go-rabbitmq"
)

func TestPostSubscribe(t *testing.T) {
	// Initialize the test server
	testCases := []struct {
		name               string
		body               string
		wantCode           int
		wantBody           string
		wantDataInDatabase map[string]interface{}
	}{
		{
			name:     "valid subscription",
			body:     "{\"email\":\"test@gmail.com\",\"city\":\"Kyiv\",\"frequency\":\"hourly\"}",
			wantCode: http.StatusOK,
			wantBody: `{"message":"Subscribed successfully"}`,
			wantDataInDatabase: map[string]interface{}{
				"email":     "test@gmail.com",
				"city":      "Kyiv",
				"frequency": "hourly",
				"Count":     1,
			},
		},
		{
			name:     "email and city already subscribed",
			body:     "{\"email\":\"test@gmail.com\",\"city\":\"Kyiv\",\"frequency\":\"hourly\"}",
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"Email and city already subscribed"}`,
			wantDataInDatabase: map[string]interface{}{
				"email":     "test@gmail.com",
				"city":      "Kyiv",
				"frequency": "hourly",
				"Count":     1,
			},
		},
	}

	err := resetTables(db)

	assert.NoError(t, err)

	consumer, err := rabbitmq.NewConsumer(
		rmqConn,
		messaging.SubscribeQueueName,
		rabbitmq.WithConsumerOptionsExchangeName(messaging.ExchangeName),
		rabbitmq.WithConsumerOptionsExchangeDeclare,
		rabbitmq.WithConsumerOptionsExchangeDurable,
		rabbitmq.WithConsumerOptionsRoutingKey(messaging.SubscribeRoutingKey),
		rabbitmq.WithConsumerOptionsQueueDurable,
		// rabbitmq.WithConsumerOptionsQueueDe,
	)

	require.NoError(t, err)
	defer consumer.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log.Printf("data to send: %s", tc.body)

			// Create a new request with context
			var req *http.Request
			ctx := context.Background()
			req, err = http.NewRequestWithContext(ctx, http.MethodPost,
				testServerURL+"/subscribe", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			time.Sleep(5000 * time.Millisecond)

			defer func(body io.ReadCloser) {
				err := body.Close()
				assert.NoError(t, err, "Failed to close response body")
			}(resp.Body)

			// Check the status code
			assert.Equalf(t, tc.wantCode, resp.StatusCode,
				"Expected status code %d, got %d", tc.wantCode, resp.StatusCode)

			// Check the response body
			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body")

			bodyString := string(bodyBytes)

			// Check if the body matches the expected response
			assert.Equalf(t, bodyString, tc.wantBody, "Expected body %s, got %s", tc.wantBody, bodyString)

			subscription := FetchSubscription(t, tc.wantDataInDatabase["email"].(string), //nolint:errcheck
				tc.wantDataInDatabase["city"].(string)) //nolint:errcheck

			// Check if the subscription was created in the database
			assert.NotNil(t, subscription, "Expected subscription to be created")
			assert.Equal(t, tc.wantDataInDatabase["email"], subscription["email"], "Expected email to match")
			assert.Equal(t, tc.wantDataInDatabase["city"], subscription["city"], "Expected city to match")
			assert.Equal(t, tc.wantDataInDatabase["frequency"], subscription["frequency"],
				"Expected frequency to match")
			assert.Equal(t, tc.wantDataInDatabase["Count"], subscription["Count"], "Expected Count to match")

			// check if the new subscription was sent to the RabbitMQ queue
			msg, err := readLatestRabbitMQMessage(consumer, messaging.SubscribeQueueName)
			require.NoError(t, err, "Failed to read message from RabbitMQ queue")

			require.NotNil(t, msg, "Expected a message in the RabbitMQ queue")
			var event messaging.NewSubscriptionEvent
			err = json.Unmarshal(msg, &event)
			require.NoError(t, err, "Failed to unmarshal RabbitMQ message")
			assert.Equal(t,
				tc.wantDataInDatabase["email"],
				event.Email,
				"Expected email to match in RabbitMQ message")
			assert.Equal(t,
				tc.wantDataInDatabase["token"],
				event.Token,
				"Expected city to match in RabbitMQ message")
		},
		)
	}
}
