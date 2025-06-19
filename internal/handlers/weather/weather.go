package weather

import (
	"context"
	"net/http"

	service "github.com/Nazarious-ucu/weather-subscription-api/internal/services"

	"github.com/gin-gonic/gin"
)

type WeatherServicer interface {
	GetByCity(ctx context.Context, city string) (service.WeatherData, error)
}

type Handler struct {
	Service WeatherServicer
}

func NewHandler(svc WeatherServicer) *Handler {
	return &Handler{Service: svc}
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
func (h *Handler) GetWeather(c *gin.Context) {
	city := c.Query("city")
	if city == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "city query parameter is required"})
		return
	}
	ctx := context.Background()

	data, err := h.Service.GetByCity(ctx, city)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}
