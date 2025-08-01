package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/metrics"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const timeoutDuration = 10 * time.Second

var ErrSubscriptionExists = errors.New("subscription already exists")

// Handler handles subscription HTTP endpoints with structured logging and metrics.
type Handler struct {
	svc    Service
	logger zerolog.Logger
	m      *metrics.Metrics
}

// Service defines the subscription business operations.
type Service interface {
	Subscribe(ctx context.Context, data models.UserSubData) error
	Confirm(ctx context.Context, token string) (bool, error)
	Unsubscribe(ctx context.Context, token string) (bool, error)
}

// NewHandler creates an HTTP handler with structured logging and metrics.
func NewHandler(svc Service, logger zerolog.Logger, m *metrics.Metrics) *Handler {
	logger = logger.With().Str("component", "SubscriptionHTTPHandler").Logger()
	return &Handler{svc: svc, logger: logger, m: m}
}

// Subscribe handles POST /subscribe requests.
// Subscribe handles POST /subscribe requests.
func (h *Handler) Subscribe(c *gin.Context) {
	start := time.Now()
	var userData models.UserSubData
	if err := c.ShouldBindJSON(&userData); err != nil {
		h.logger.Warn().Err(err).
			Str("error_type", "bind_error").
			Dur("duration", time.Since(start)).
			Msg("Missing required fields")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutDuration)
	defer cancel()

	err := h.svc.Subscribe(ctx, userData)
	if err != nil {
		if errors.Is(err, ErrSubscriptionExists) {
			h.logger.Warn().Err(err).
				Str("error_type", "duplicate_subscription").
				Dur("duration", time.Since(start)).
				Msg("Subscription already exists")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email and city already subscribed"})
			return
		}
		h.logger.Error().Err(err).
			Str("error_type", "subscribe_error").
			Dur("duration", time.Since(start)).
			Msg("Subscription failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Business metric
	h.m.SubscriptionsCreated.WithLabelValues(userData.Frequency).Inc()
	h.logger.Info().
		Str("email", userData.Email).
		Str("city", userData.City).
		Str("frequency", userData.Frequency).
		Dur("duration", time.Since(start)).
		Msg("Subscription created")

	// success response
	c.JSON(http.StatusOK, gin.H{"message": "Subscribed successfully"})
}

// Confirm handles GET /confirm/:token requests.
func (h *Handler) Confirm(c *gin.Context) {
	start := time.Now()
	token := c.Param("token")
	h.logger.Debug().Str("token", token).Msg("Confirm called")

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutDuration)
	defer cancel()

	ok, err := h.svc.Confirm(ctx, token)
	if err != nil {
		h.logger.Error().Err(err).
			Str("error_type", "confirm_error").
			Dur("duration", time.Since(start)).
			Msg("Confirmation failed")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !ok {
		h.logger.Warn().
			Str("error_type", "invalid_token").
			Dur("duration", time.Since(start)).
			Msg("Invalid confirmation token")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Business metric
	h.m.SubscriptionsConfirmed.Inc()
	h.logger.Info().Str("token", token).
		Dur("duration", time.Since(start)).
		Msg("Subscription confirmed")
	c.Status(http.StatusOK)
}

// Unsubscribe handles GET /unsubscribe/:token requests.
func (h *Handler) Unsubscribe(c *gin.Context) {
	start := time.Now()
	token := c.Param("token")
	h.logger.Debug().Str("token", token).Msg("Unsubscribe called")

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutDuration)
	defer cancel()

	ok, err := h.svc.Unsubscribe(ctx, token)
	if err != nil {
		h.logger.Error().Err(err).
			Str("error_type", "unsubscribe_error").
			Dur("duration", time.Since(start)).
			Msg("Unsubscription failed")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !ok {
		h.logger.Warn().
			Str("error_type", "invalid_token").
			Dur("duration", time.Since(start)).
			Msg("Invalid unsubscribe token")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Business metric
	h.m.SubscriptionsCanceled.Inc()
	h.logger.Info().Str("token", token).
		Dur("duration", time.Since(start)).
		Msg("Subscription canceled")
	c.Status(http.StatusOK)
}
