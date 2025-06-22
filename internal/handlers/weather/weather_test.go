package weather_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/weather"
)

type mockService struct {
	data models.WeatherData
	err  error
}

func (m *mockService) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	return m.data, m.err
}

func TestGetWeather_NoCity(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	var err error
	c.Request, err = http.NewRequest(http.MethodGet, "/weather", nil)
	assert.NoError(t, err)

	h := weather.NewHandler(&mockService{})
	h.GetWeather(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"city query parameter is required"}`, rec.Body.String())
}

func TestGetWeather_ServiceError(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req, err := http.NewRequest(http.MethodGet, "/weather?city=Kyiv", nil)
	assert.NoError(t, err)

	c.Request = req

	errMsg := "service unavailable"
	m := &mockService{err: errors.New(errMsg)}
	h := weather.NewHandler(m)

	h.GetWeather(c)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, fmt.Sprintf(`{"error":"%s"}`, errMsg), rec.Body.String())
}

func TestGetWeather_Success(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	data := models.WeatherData{City: "Kyiv", Temperature: 20.5, Condition: "Sunny"}

	m := &mockService{data: data}
	h := weather.NewHandler(m)

	req, err := http.NewRequest(http.MethodGet, "/weather?city=Kyiv", nil)
	assert.NoError(t, err)

	c.Request = req

	h.GetWeather(c)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, fmt.Sprintf(`{"city":"%s","temperature":%v,"condition":"%s"}`,
		data.City, data.Temperature, data.Condition), rec.Body.String())
}
