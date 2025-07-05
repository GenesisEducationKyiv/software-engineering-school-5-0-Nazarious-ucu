//go:build unit

package weather_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/weather"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	args := m.Called(ctx, city)

	data, ok := args.Get(0).(models.WeatherData)

	if !ok {
		return models.WeatherData{}, args.Error(1)
	}

	return data, args.Error(1)
}

func TestGetWeather_NoCity(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	m := &mockService{}

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	req, err := http.NewRequest(http.MethodGet, "/weather", nil)
	require.NoError(t, err)

	c.Request = req

	h := weather.NewHandler(m)

	h.GetWeather(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"city query parameter is required"}`, rec.Body.String())
}

func TestGetWeather_ServiceError(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	m := &mockService{}

	m.On("GetByCity", mock.Anything, mock.Anything).
		Return(models.WeatherData{}, errors.New("service unavailable")).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	req, err := http.NewRequest(http.MethodGet, "/weather?city=Kyiv", nil)
	assert.NoError(t, err)

	c.Request = req

	h := weather.NewHandler(m)

	h.GetWeather(c)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"error":"service unavailable"}`, rec.Body.String())
}

func TestGetWeather_Success(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	data := models.WeatherData{City: "Kyiv", Temperature: 20.5, Condition: "Sunny"}

	m := &mockService{}
	m.On("GetByCity", mock.Anything, mock.Anything).Return(data, nil).Once()

	t.Cleanup(func() {
		m.AssertExpectations(t)
	})

	h := weather.NewHandler(m)

	req, err := http.NewRequest(http.MethodGet, "/weather?city=Kyiv", nil)
	assert.NoError(t, err)

	c.Request = req

	h.GetWeather(c)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, fmt.Sprintf(`{"city":"%s","temperature":%v,"condition":"%s"}`,
		data.City, data.Temperature, data.Condition), rec.Body.String())
}
