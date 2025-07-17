package weather_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/services/weather"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var breakerCfg = weather.BreakerConfig{
	TimeInterval: 30 * time.Second,
	TimeTimeOut:  15 * time.Second,
	RepeatNumber: 5,
}

type mockWrapped struct {
	mock.Mock
}

func (m *mockWrapped) Fetch(ctx context.Context, city string) (models.WeatherData, error) {
	args := m.Called(ctx, city)
	data, ok := args.Get(0).(models.WeatherData)
	if !ok {
		return models.WeatherData{}, args.Error(1)
	}
	return data, args.Error(1)
}

const (
	breakerName = "TestAPI"
	city        = "Lviv"
)

func TestBreakerClient_Success(t *testing.T) {
	wrapped := new(mockWrapped)
	expected := models.WeatherData{City: city, Temperature: 20, Condition: "Clear"}

	wrapped.
		On("Fetch", mock.Anything, city).
		Return(expected, nil).
		Once()

	bc := weather.NewBreakerClient(breakerName, breakerCfg, wrapped)

	data, err := bc.Fetch(context.Background(), city)
	assert.NoError(t, err)
	assert.Equal(t, expected, data)

	wrapped.AssertExpectations(t)
	wrapped.AssertNumberOfCalls(t, "Fetch", 1)
}

func TestBreakerClient_UnderlyingErrorBeforeTrip(t *testing.T) {
	wrapped := new(mockWrapped)
	underlyingErr := errors.New("service down")

	wrapped.
		On("Fetch", mock.Anything, city).
		Return(models.WeatherData{}, underlyingErr).
		Once()

	bc := weather.NewBreakerClient(breakerName, breakerCfg, wrapped)

	data, err := bc.Fetch(context.Background(), city)
	assert.Error(t, err)
	assert.Empty(t, data)
	assert.Contains(t, err.Error(), breakerName+" unavailable: "+underlyingErr.Error())

	wrapped.AssertExpectations(t)
	wrapped.AssertNumberOfCalls(t, "Fetch", 1)
}

func TestBreakerClient_TripCircuitAfterFiveFailures(t *testing.T) {
	wrapped := new(mockWrapped)
	underlyingErr := errors.New("timeout")

	for i := 0; i < 5; i++ {
		wrapped.
			On("Fetch", mock.Anything, city).
			Return(models.WeatherData{}, underlyingErr).
			Once()
	}

	bc := weather.NewBreakerClient(breakerName, breakerCfg, wrapped)

	for i := 1; i <= 5; i++ {
		_, err := bc.Fetch(context.Background(), city)
		assert.Error(t, err, "call #%d should error before trip", i)
		assert.Contains(t, err.Error(), breakerName+" unavailable: "+underlyingErr.Error())
	}

	_, err := bc.Fetch(context.Background(), city)
	assert.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "circuit breaker is open"),
		"6th call should return open-circuit error",
	)

	wrapped.AssertExpectations(t)
	wrapped.AssertNumberOfCalls(t, "Fetch", 5)
}
