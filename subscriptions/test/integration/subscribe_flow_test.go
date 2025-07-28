//go:build integration

package integration

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

			defer func(body io.ReadCloser) {
				err := body.Close()
				assert.NoError(t, err, "Failed to close response body")
			}(resp.Body)

			// Check the status code
			assert.Equalf(t, tc.wantCode, resp.StatusCode,
				"Expected status code %d, got %d", tc.wantCode, resp.StatusCode)

			// Check the response body
			bodyBytes, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "Failed to read response body")

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

			// send request to smtp server to check if email was sent
			req, err = http.NewRequestWithContext(ctx, http.MethodGet,
				"http://localhost:8025/api/v2/messages", nil)
			assert.NoError(t, err, "Failed to create request to SMTP server")

			resp, err = http.DefaultClient.Do(req)
			assert.NoError(t, err, "Failed to get messages from SMTP server")
			defer func(body io.ReadCloser) {
				err := body.Close()
				assert.NoError(t, err, "Failed to close response body")
			}(resp.Body)

			// Check was the email sent to the SMTP server
			assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200 from SMTP server")
			bodyBytes, err = io.ReadAll(resp.Body)
			assert.NoError(t, err, "Failed to read response body from SMTP server")
			bodyString = string(bodyBytes)
			assert.Contains(t, bodyString, tc.wantDataInDatabase["email"].(string), //nolint:errcheck
				"Expected email to be sent to the SMTP server")
		})
	}
}
