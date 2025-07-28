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

func Test_WeatherBit_GetByCity_Success(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)

	m := &mockHTTPClient{}

	m.On("Do", mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{
					  "data": [
						{
						  "city_name": "Odesa",
						  "temp": 27.5,
						  "weather": {
							"description": "sunny"
						  }
						}
					  ]
					}`)),
		}, nil).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	weatherAPIClient := weather.NewClientWeatherBit("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "Odesa")
	assert.NoError(t, err)
	assert.Equal(t, "Odesa", data.City)
	assert.Equal(t, 27.5, data.Temperature)
	assert.Equal(t, "sunny", data.Condition)
}

func Test_WeatherBit_CityNotFound(t *testing.T) {
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

	weatherAPIClient := weather.NewClientWeatherBit("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "UnknownCity")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func Test_WeatherBit_APIError(t *testing.T) {
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

	weatherAPIClient := weather.NewClientWeatherBit("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}

func Test_WeatherBit_GetByCity_InvalidAPIKey(t *testing.T) {
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

	weatherAPIClient := weather.NewClientWeatherBit("1234567890", "", m, log.Default())

	data, err := weatherAPIClient.Fetch(ctx, "London")
	assert.Error(t, err)
	assert.Equal(t, models.WeatherData{}, data)
}
