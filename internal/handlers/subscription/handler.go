package subscription

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"

	"github.com/gin-gonic/gin"
)

const timeoutDuration = 10 * time.Second

var ErrSubscriptionExists = errors.New("subscription already exists")

type subscriber interface {
	Subscribe(ctx context.Context, data models.UserSubData) error
	Confirm(ctx context.Context, token string) (bool, error)
	Unsubscribe(ctx context.Context, token string) (bool, error)
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
	var userData models.UserSubData
	if err := c.ShouldBind(&userData); err != nil {
		log.Printf("Failed to bind user data: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutDuration)
	defer cancel()

	err := h.Service.Subscribe(ctx, userData)
	if err != nil {
		if errors.Is(err, ErrSubscriptionExists) {
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutDuration)
	defer cancel()

	ok, err := h.Service.Confirm(ctx, token)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !ok {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusOK)
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutDuration)
	defer cancel()

	ok, err := h.Service.Unsubscribe(ctx, token)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !ok {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusOK)
}
