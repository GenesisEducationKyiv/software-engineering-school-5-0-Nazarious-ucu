//go:build integration

package integration

import (
	"context"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWeatherFlow(t *testing.T) {
	testCases := []struct {
		name     string
		city     string
		wantCode int
		wantBody string
	}{
		{
			name:     "valid city",
			city:     "Kyiv",
			wantCode: http.StatusOK,
			wantBody: `{"temperature":10000.0,"condition":"Sunny","city":"H_E_L_L"}`,
		},
		{
			name:     "invalid city",
			city:     "InvalidCity",
			wantCode: http.StatusNotFound,
			wantBody: `{"error":"City not found"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log.Printf("city to send: %s", tc.city)

			// Create HTTP GET request to test server URL
			url := testServerURL + "/api/weather?city=" + tc.city
			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				url,
				nil,
			)
			assert.NoError(t, err)

			// Send the request using the test server
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			defer func(body io.ReadCloser) {
				err := body.Close()
				assert.NoError(t, err, "Failed to close response body")
			}(resp.Body)

			// Check the response status code
			assert.Equal(t, tc.wantCode, resp.StatusCode)

			// Read the response body
			bodyBytes, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "failed reading response body")
			assert.JSONEq(t, tc.wantBody, string(bodyBytes))
		})
	}
}
