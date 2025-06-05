package handlers

import (
	_ "WeatherSubscriptionAPI/internal/models"
	service "WeatherSubscriptionAPI/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	Service *service.SubscriptionService
}

func NewSubscriptionHandler(svc *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{Service: svc}
}

// Subscribe
// @Summary Subscribe to weather updates
// @Description Subscribe an email to receive weather updates for a specific city.
// @Tags subscription
// @Accept application/x-www-form-urlencoded
// @Param email formData string true "Email address to subscribe"
// @Param city formData string true "City for weather updates"
// @Param frequency formData string true "Frequency of updates" Enums(hourly, daily)
// @Success 200
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /subscribe [post]
func (h *SubscriptionHandler) Subscribe(c *gin.Context) {
	email := c.PostForm("email")
	city := c.PostForm("city")
	frequency := c.PostForm("frequency")
	if email == "" || city == "" {
		c.Writer.WriteHeader(http.StatusBadRequest)
		return
	}
	err := h.Service.Subscribe(email, city, frequency)
	if err != nil {
		//c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	//c.JSON(http.StatusOK, gin.H{"message": "Subscription successful.", "token": token})
	c.Writer.WriteHeader(http.StatusOK)
}

// Confirm
// @Summary Confirm subscription
// @Description Confirms the subscription using the token sent in email.
// @Tags subscription
// @Param token path string true "Confirmation token"
// @Success 200
// @Failure 400
// @Failure 404
// @Router /confirm/{token} [get]
func (h *SubscriptionHandler) Confirm(c *gin.Context) {
	token := c.Param("token")
	ok, err := h.Service.Confirm(token)
	if err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		c.Writer.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Writer.WriteHeader(http.StatusOK)
}

// Unsubscribe
// @Summary Unsubscribe
// @Description Unsubscribe from weather updates using the token.
// @Tags subscription
// @Param token path string true "Unsubscribe token"
// @Success 200
// @Failure 400
// @Failure 404
// @Router /unsubscribe/{token} [get]
func (h *SubscriptionHandler) Unsubscribe(c *gin.Context) {
	token := c.Param("token")
	ok, err := h.Service.Unsubscribe(token)
	if err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		c.Writer.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Writer.WriteHeader(http.StatusOK)
}
