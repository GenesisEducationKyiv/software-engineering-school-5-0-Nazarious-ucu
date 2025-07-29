//go:build integration

package api

import (
	"context"
	"log"
	"testing"
	"time"

	"google.golang.org/grpc/credentials/insecure"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	weatherpb "github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/weather"
)

func TestWeatherService_GetByCity(t *testing.T) {
	conn, err := grpc.NewClient(
		testServerURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithIdleTimeout(5*time.Second))
	require.NoError(t, err, "failed to connect to weather gRPC service")
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Printf("failed to close gRPC connection: %v", err)
		} else {
			log.Println("gRPC connection closed successfully")
		}
	}(conn)

	client := weatherpb.NewWeatherServiceClient(conn)

	testCases := []struct {
		name     string
		city     string
		wantResp *weatherpb.WeatherResponse
		wantErr  string
	}{
		{
			name: "valid city",
			city: "H_E_L_L",
			wantResp: &weatherpb.WeatherResponse{
				City:        "H_E_L_L",
				Temperature: 10000.0,
				Condition:   "Sunny",
			},
			wantErr: "",
		},
		{
			name:     "invalid city",
			city:     "InvalidCity",
			wantResp: nil,
			wantErr:  "all weather API clients failed to fetch data",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log.Printf("Requesting weather for city: %s", tc.city)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			resp, err := client.GetByCity(ctx, &weatherpb.WeatherRequest{City: tc.city})

			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tc.wantResp.City, resp.City)
				assert.Equal(t, tc.wantResp.Condition, resp.Condition)
				assert.Equal(t, tc.wantResp.Temperature, resp.Temperature)
			}
		})
	}
}
