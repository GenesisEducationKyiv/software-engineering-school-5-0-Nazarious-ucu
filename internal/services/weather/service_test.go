package weather

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAPIClient struct {
	mock.Mock
}

func (m *mockAPIClient) Fetch(
	ctx context.Context,
	city string,
) (models.WeatherData, error) {
	args := m.Called(ctx, city)
	data, ok := args.Get(0).(models.WeatherData)

	if !ok {
		return models.WeatherData{}, args.Error(1)
	}

	return data, args.Error(1)
}

func TestServiceProvider_GetByCity(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)
	successWeatherModel := models.WeatherData{City: "Lviv", Temperature: 15, Condition: "Sunny"}
	emptyModel := models.WeatherData{}

	discardLogger := log.New(io.Discard, "", 0)

	t.Run("Success", func(t *testing.T) {
		mock1 := mockAPIClient{}
		mock2 := mockAPIClient{}

		mock1.On("Fetch", mock.Anything, "Lviv").Return(successWeatherModel, nil)

		t.Cleanup(func() {
			mock1.AssertExpectations(t)
			mock2.AssertNumberOfCalls(t, "Fetch", 0)
		})

		provider := NewService(discardLogger, &mock1, &mock2)

		result, err := provider.GetByCity(ctx, "Lviv")

		require.NoError(t, err)

		assert.Equal(t, successWeatherModel, result)
	})

	t.Run("FirstFailsSecondSuccess", func(t *testing.T) {
		mock1 := mockAPIClient{}
		mock2 := mockAPIClient{}

		mock1.On("Fetch", mock.Anything, "Lviv").Return(emptyModel, errors.New("error"))
		mock2.On("Fetch", mock.Anything, "Lviv").Return(successWeatherModel, nil)

		t.Cleanup(func() {
			mock1.AssertExpectations(t)
			mock2.AssertExpectations(t)
		})
		provider := NewService(discardLogger, &mock1, &mock2)

		result, err := provider.GetByCity(ctx, "Lviv")

		require.NoError(t, err)

		assert.Equal(t, successWeatherModel, result)
	})

	t.Run("AllFails", func(t *testing.T) {
		mock1 := mockAPIClient{}
		mock2 := mockAPIClient{}

		mock1.On("Fetch", mock.Anything, "Lviv").Return(emptyModel, errors.New("error"))
		mock2.On("Fetch", mock.Anything, "Lviv").Return(emptyModel, errors.New("error"))

		t.Cleanup(func() {
			mock1.AssertExpectations(t)
			mock2.AssertExpectations(t)
		})

		provider := NewService(discardLogger, &mock1, &mock2)

		result, err := provider.GetByCity(ctx, "Lviv")

		require.Error(t, err)
		assert.Equal(t, err.Error(), "all weather API clients failed to fetch data")
		assert.Equal(t, emptyModel, result)
	})
}
