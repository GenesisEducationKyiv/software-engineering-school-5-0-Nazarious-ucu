//go:build integration

package integration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

const bytesNum = 16

func TestPostConfirm(t *testing.T) {
	err := resetTables(db)

	assert.NoError(t, err)

	tokenBytes := make([]byte, bytesNum)
	_, err = rand.Read(tokenBytes)
	assert.NoError(t, err, "Failed to generate random token bytes")

	token := hex.EncodeToString(tokenBytes)

	testCases := []struct {
		name          string
		token         string
		wantCode      int
		wantConfirmed bool
	}{
		{
			name:          "invalid token",
			token:         "invalid-token",
			wantCode:      http.StatusBadRequest,
			wantConfirmed: false,
		},
		{
			name:          "valid confirmation",
			token:         token,
			wantCode:      http.StatusOK,
			wantConfirmed: true,
		},

		//	{
		//	name:     "already confirmed",
		//	token:    token,
		//	wantCode: http.StatusBadRequest,
		//	wantBody: `{"error":"Subscription already confirmed"}`,
		//	},

	}

	saveSubscription(t, "test2@gmail.com", "Lviv", "hourly", token)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log.Printf("Test: %s, with token: %s", tc.name, tc.token)

			ctx := context.Background()
			// Create a new HTTP GET request
			req, err := http.NewRequestWithContext(ctx,
				http.MethodGet, testServerURL+"/confirm/"+tc.token, nil)
			assert.NoError(t, err)

			// Perform the request
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			defer func(body io.ReadCloser) {
				err := body.Close()
				assert.NoError(t, err, "Failed to close response body")
			}(resp.Body)

			// Check the response status code
			assert.Equal(t, tc.wantCode, resp.StatusCode)

			// Check a Database for the subscription status
			var confirmed bool
			err = db.QueryRowContext(ctx,
				"SELECT confirmed FROM subscriptions WHERE token = $1", token).Scan(&confirmed)
			require.NoError(t, err, "Failed to query subscription status")
			assert.Equal(t, tc.wantConfirmed, confirmed, "Subscription should be confirmed")
		})
	}
}
