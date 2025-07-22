package weather

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"

	"github.com/gin-gonic/gin"
)

const timeoutDuration = 10 * time.Second

type weatherGetterService interface {
	GetByCity(ctx context.Context, city string) (models.WeatherData, error)
}

type Handler struct {
	service weatherGetterService
}

func NewHandler(svc weatherGetterService) *Handler {
	return &Handler{service: svc}
}

// GetWeather
// @Summary Get current weather
// @Description Returns the current weather for a given city
// @Tags weather
// @Accept json
// @Produce json
// @Param city query string true "City name"
// @Success 200 {object} models.WeatherData
// @Failure 400
// @Failure 500
// @Router /weather [get]
func (h *Handler) GetWeather(c *gin.Context) {
	city := c.Query("city")
	if city == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "city query parameter is required"})
		return
	}
	ctxWithTimeout, cancel := context.WithTimeout(c.Request.Context(), timeoutDuration)
	defer cancel()

	data, err := h.service.GetByCity(ctxWithTimeout, city)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			c.JSON(http.StatusNotFound, gin.H{"error": "City not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}
