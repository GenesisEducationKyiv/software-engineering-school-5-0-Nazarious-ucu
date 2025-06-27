package weather_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/weather"
	"github.com/stretchr/testify/assert"
)

type mockHTTPClient struct {
	mock.Mock
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	resp, err := args.Get(0).(*http.Response)
	if err == false {
		return &http.Response{}, args.Error(1)
	}
	return resp, args.Error(1)
}

func TestGetByCity_Success(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)

	m := &mockHTTPClient{}

	m.On("Do", mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"location": {"name": "London"},
						"current": {"temp_c": 15.0, "condition": {"text": "Sunny"}}}`)),
		}, nil).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	weatherService := weather.NewService("mock_api_key", m, &log.Logger{},
		"https://api.weatherapi.com/v1/current.json")

	data, err := weatherService.GetByCity(ctx, "London")
	assert.NoError(t, err)
	assert.Equal(t, "London", data.City)
	assert.Equal(t, 15.0, data.Temperature)
	assert.Equal(t, "Sunny", data.Condition)
}

func TestGetByCity_CityNotFound(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)

	m := &mockHTTPClient{}

	m.On("Do", mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader(`{"error": "City not found"}`)),
		}, nil).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	weatherService := weather.NewService("mock_api_key",
		m, &log.Logger{}, "https://api.weatherapi.com/v1/current.json")

	data, err := weatherService.GetByCity(ctx, "UnknownCity")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func TestGetByCity_APIError(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)

	m := &mockHTTPClient{}

	m.On("Do", mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(`{"error": "Internal server error"}`)),
		}, nil).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	weatherService := weather.NewService("mock_api_key",
		m, &log.Logger{}, "https://api.weatherapi.com/v1/current.json")

	data, err := weatherService.GetByCity(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func TestGetByCity_InvalidAPIKey(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)

	m := &mockHTTPClient{}

	m.On("Do", mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{"error": "Invalid API key"}`)),
		}, nil).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	weatherService := weather.NewService("invalid_api_key",
		m, &log.Logger{}, "https://api.weatherapi.com/v1/current.json")

	data, err := weatherService.GetByCity(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func TestGetByCity_Timeout(t *testing.T) {
	testCtx, _ := gin.CreateTestContext(nil)

	ctx, cancel := context.WithTimeout(testCtx, 0)
	defer cancel()
	m := &mockHTTPClient{}

	m.On("Do", mock.Anything).Return(nil, ctx.Err())
	t.Cleanup(func() {
		m.AssertExpectations(t)
	})
	weatherService := weather.NewService("mock_api_key",
		m, &log.Logger{}, "https://api.weatherapi.com/v1/current.json")

	data, err := weatherService.GetByCity(ctx, "London")
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, models.WeatherData{}, data)
}
