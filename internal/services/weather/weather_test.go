package weather_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/weather"
	"github.com/stretchr/testify/assert"
)

type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func TestGetByCity(t *testing.T) {
	mockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://api.weatherapi.com/v1/current.json?key=mock_api_key&q=London" {
				return nil, fmt.Errorf("unexpected URL: %s", req.URL.String())
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"location": {"name": "London"}, 
						"current": {"temp_c": 15.0, "condition": {"text": "Sunny"}}}`)),
			}, nil
		},
	}

	weatherService := weather.NewService("mock_api_key", mockClient, &log.Logger{},
		"https://api.weatherapi.com/v1/current.json")

	ctx := context.Background()
	data, err := weatherService.GetByCity(ctx, "London")
	assert.NoError(t, err)
	assert.Equal(t, "London", data.City)
	assert.Equal(t, 15.0, data.Temperature)
	assert.Equal(t, "Sunny", data.Condition)
}

func TestGetByCity_CityNotFound(t *testing.T) {
	mockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`{"error": "City not found"}`)),
			}, nil
		},
	}

	weatherService := weather.NewService("mock_api_key",
		mockClient, &log.Logger{}, "https://api.weatherapi.com/v1/current.json")

	ctx := context.Background()
	data, err := weatherService.GetByCity(ctx, "UnknownCity")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func TestGetByCity_APIError(t *testing.T) {
	mockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`{"error": "Internal server error"}`)),
			}, nil
		},
	}

	weatherService := weather.NewService("mock_api_key",
		mockClient, &log.Logger{}, "https://api.weatherapi.com/v1/current.json")

	ctx := context.Background()
	data, err := weatherService.GetByCity(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func TestGetByCity_InvalidAPIKey(t *testing.T) {
	mockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"error": "Invalid API key"}`)),
			}, nil
		},
	}
	weatherService := weather.NewService("invalid_api_key",
		mockClient, &log.Logger{}, "https://api.weatherapi.com/v1/current.json")

	ctx := context.Background()
	data, err := weatherService.GetByCity(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func TestGetByCity_Timeout(t *testing.T) {
	mockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			select {
			case <-time.After(2 * time.Second):
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"location": {"name": "London"}, "current": 
							{"temp_c": 15.0, "condition": {"text": "Sunny"}}}`)),
				}, nil
			case <-time.After(1 * time.Second):
				return nil, errors.New("request timed out")
			}
		},
	}
	weatherService := weather.NewService("mock_api_key",
		mockClient, &log.Logger{}, "https://api.weatherapi.com/v1/current.json")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	data, err := weatherService.GetByCity(ctx, "London")
	assert.Equal(t, errors.New("request timed out"), err)
	assert.Equal(t, models.WeatherData{}, data)
}
