package handlers

import (
	"WeatherSubscriptionAPI/internal/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

type WeatherHandler struct {
	Service *service.WeatherService
}

func NewWeatherHandler(svc *service.WeatherService) *WeatherHandler {
	return &WeatherHandler{Service: svc}
}

// GetWeather
// @Summary Get current weather
// @Description Returns the current weather for a given city
// @Tags weather
// @Accept json
// @Produce json
// @Param city query string true "City name"
// @Success 200 {object} service.WeatherData
// @Failure 400
// @Failure 500
// @Router /weather [get]
func (h *WeatherHandler) GetWeather(c *gin.Context) {
	city := c.Query("city")
	if city == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "city query parameter is required"})
		return
	}
	data, err := h.Service.GetWeather(city)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}
