//go:build integration
// +build integration

package integration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnsubscribeFlow(t *testing.T) {
	err := resetTables(db)
	assert.NoError(t, err)

	tokenBytes := make([]byte, 16)
	_, err = rand.Read(tokenBytes)
	assert.NoError(t, err, "Failed to generate random token bytes")

	token := hex.EncodeToString(tokenBytes)

	// Initialize the test server
	testCases := []struct {
		name             string
		token            string
		wantCode         int
		wantUnsubscribed bool
	}{
		{
			name:             "invalid token",
			token:            "invalid-token",
			wantCode:         http.StatusBadRequest,
			wantUnsubscribed: false,
		},
		{
			name:             "valid unsubscription",
			token:            token,
			wantCode:         http.StatusOK,
			wantUnsubscribed: true,
		},
	}

	saveSubscription(t, "test3@gmail.com", "Odesa", "daily", token)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log.Printf("token to send: %s", tc.token)

			// Create HTTP GET request to test server URL

			url := testServerURL + "/api/unsubscribe/" + tc.token
			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				url,
				nil,
			)
			assert.NoError(t, err)

			q := req.URL.Query()
			q.Add("token", tc.token)
			req.URL.RawQuery = q.Encode()

			// Send the request
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)

			defer func(body io.ReadCloser) {
				err := body.Close()
				if err != nil {
					log.Printf("Error closing response body: %v", err)
				} else {
					log.Println("Response body closed successfully")
				}
			}(resp.Body)

			// Check the response status code
			assert.Equal(t, tc.wantCode, resp.StatusCode)

			// Check if the unsubscription was successful
			if tc.wantUnsubscribed {
				unsubscribed, err := unsubscribe(tc.token)
				assert.NoError(t, err)
				assert.True(t, unsubscribed)
			} else {
				unsubscribed, err := unsubscribe(tc.token)
				assert.Error(t, err)
				assert.False(t, unsubscribed)
			}
		})
	}
}

func unsubscribe(token string) (bool, error) {
	// parse the subscription by token and check if it unsubscribed
	var unsubscribed bool
	err := db.QueryRowContext(context.Background(),
		"UPDATE subscriptions "+
			"SET unsubscribed = TRUE WHERE token = $1 RETURNING unsubscribed", token).Scan(&unsubscribed)
	if err != nil {
		return false, err
	}
	return unsubscribed, nil
}
