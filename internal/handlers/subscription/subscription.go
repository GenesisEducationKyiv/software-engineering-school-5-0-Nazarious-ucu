package subscription

import (
	"log"
	"net/http"

	_ "github.com/Nazarious-ucu/weather-subscription-api/internal/models"

	"github.com/gin-gonic/gin"
)

type subscriber interface {
	Subscribe(email, city, frequency string) error
	Confirm(token string) (bool, error)
	Unsubscribe(token string) (bool, error)
}

type SubscriptionHandler struct {
	Service subscriber
}

func NewHandler(svc subscriber) *SubscriptionHandler {
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
	log.Printf("email: %s, city: %s, frequency: %s",
		c.PostForm("email"), c.PostForm("city"), c.PostForm("frequency"))
	email := c.PostForm("email")
	city := c.PostForm("city")
	frequency := c.PostForm("frequency")
	if email == "" || city == "" || frequency == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}
	err := h.Service.Subscribe(email, city, frequency)
	if err != nil {
		if err.Error() == "subscription already exists" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email and city already subscribed"})
			return
		}
		log.Printf("Failed to subscribe with that error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscribed successfully"})
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
