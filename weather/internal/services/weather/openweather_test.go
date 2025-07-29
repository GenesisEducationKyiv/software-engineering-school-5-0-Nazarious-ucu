//go:build unit

package weather_test

import (
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/weather"
	"github.com/stretchr/testify/assert"
)

func Test_OpenWeather_GetByCity_Success(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)

	m := &mockHTTPClient{}

	m.On("Do", mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{
				  "main": {
					"temp": 15.0,
					"feels_like": 24.0,
					"pressure": 1013,
					"humidity": 60
				  },
				  "weather": [
					{
					  "main": "Sunny",
					  "description": "Cool"
					}
				  ]
				}`)),
		}, nil).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	weatherAPIClient := weather.NewClientOpenWeatherMap("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "London")
	assert.NoError(t, err)
	assert.Equal(t, "London", data.City)
	assert.Equal(t, 15.0, data.Temperature)
	assert.Equal(t, "Sunny", data.Condition)
}

func Test_OpenWeatherGetByCity_CityNotFound(t *testing.T) {
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

	weatherAPIClient := weather.NewClientWeatherAPI("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "UnknownCity")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func Test_OpenWeatherGetByCity_APIError(t *testing.T) {
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

	weatherAPIClient := weather.NewClientWeatherAPI("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func Test_OpenWeatherGetByCity_InvalidAPIKey(t *testing.T) {
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

	weatherAPIClient := weather.NewClientWeatherAPI("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}
