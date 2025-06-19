package service_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	service "github.com/Nazarious-ucu/weather-subscription-api/internal/services"
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

	weatherService := service.NewWeatherService("mock_api_key", mockClient)

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

	weatherService := service.NewWeatherService("mock_api_key", mockClient)

	ctx := context.Background()
	data, err := weatherService.GetByCity(ctx, "UnknownCity")
	assert.Error(t, err)
	assert.Equal(t, service.WeatherData{}, data)
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

	weatherService := service.NewWeatherService("mock_api_key", mockClient)

	ctx := context.Background()
	data, err := weatherService.GetByCity(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, service.WeatherData{}, data)
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

	weatherService := service.NewWeatherService("invalid_api_key", mockClient)

	ctx := context.Background()
	data, err := weatherService.GetByCity(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, service.WeatherData{}, data)
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

	weatherService := service.NewWeatherService("mock_api_key", mockClient)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	data, err := weatherService.GetByCity(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, service.WeatherData{}, data)
}
