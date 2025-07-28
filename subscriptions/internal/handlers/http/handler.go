package http

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/models"

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
