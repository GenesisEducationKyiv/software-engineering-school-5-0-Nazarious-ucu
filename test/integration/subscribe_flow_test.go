//go:build integration
// +build integration

package integration

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostSubscribe(t *testing.T) {
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

			form := url.Values{}
			form.Set("email", tc.wantDataInDatabase["email"].(string))         //nolint:errcheck
			form.Set("city", tc.wantDataInDatabase["city"].(string))           //nolint:errcheck
			form.Set("frequency", tc.wantDataInDatabase["frequency"].(string)) //nolint:errcheck

			var req *http.Request
			ctx := context.Background()
			req, err = http.NewRequestWithContext(ctx, http.MethodPost,
				testServerURL+"api/subscribe", strings.NewReader(form.Encode()))

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)

			defer func(body io.ReadCloser) {
				err := body.Close()
				assert.NoError(t, err, "Failed to close response body")
			}(resp.Body)

			assert.Equalf(t, resp.StatusCode, tc.wantCode,
				"Expected status code %d, got %d", tc.wantCode, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "Failed to read response body")

			bodyString := string(bodyBytes)

			assert.Equalf(t, bodyString, tc.wantBody, "Expected body %s, got %s", tc.wantBody, bodyString)

			subscription := FetchSubscription(t, tc.wantDataInDatabase["email"].(string), //nolint:errcheck
				tc.wantDataInDatabase["city"].(string)) //nolint:errcheck

			assert.NotNil(t, subscription, "Expected subscription to be created")
			assert.Equal(t, tc.wantDataInDatabase["email"], subscription["email"], "Expected email to match")
			assert.Equal(t, tc.wantDataInDatabase["city"], subscription["city"], "Expected city to match")
			assert.Equal(t, tc.wantDataInDatabase["frequency"], subscription["frequency"],
				"Expected frequency to match")
			assert.Equal(t, tc.wantDataInDatabase["Count"], subscription["Count"], "Expected Count to match")
		})
	}
}
