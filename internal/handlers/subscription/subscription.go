package subscription

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserSubData struct {
	Email     string `json:"email" binding:"required,email"`
	City      string `json:"city" binding:"required"`
	Frequency string `json:"frequency" binding:"required,oneof=hourly daily"`
}

type subscriber interface {
	Subscribe(data UserSubData) error
	Confirm(token string) (bool, error)
	Unsubscribe(token string) (bool, error)
}

type Handler struct {
	Service subscriber
}

func NewHandler(svc subscriber) *Handler {
	return &Handler{Service: svc}
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
func (h *Handler) Subscribe(c *gin.Context) {
	var userData UserSubData
	if err := c.ShouldBind(&userData); err != nil {
		log.Printf("Failed to bind user data: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	err := h.Service.Subscribe(userData)
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
func (h *Handler) Confirm(c *gin.Context) {
	log.Printf("token: %s", c.Param("token"))
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
func (h *Handler) Unsubscribe(c *gin.Context) {
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
