package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/metrics"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/models"
)

const timeoutDuration = 10 * time.Second

var ErrSubscriptionExists = errors.New("subscription already exists")

type Handler struct {
	client *http.Client
	subURL string
	logger zerolog.Logger
	m      *metrics.Metrics
}

func NewHandler(
	client *http.Client,
	subscribeURL string,
	logger zerolog.Logger,
	metrics *metrics.Metrics) *Handler {
	return &Handler{
		client: client,
		subURL: subscribeURL,
		logger: logger,
		m:      metrics,
	}
}

// HandleSubscribe
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
func (h *Handler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		h.logger.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("method not allowed")

		h.m.BusinessErrors.
			WithLabelValues("method_not_allowed", "405", "warning").
			Inc()

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to parse form")

		h.m.TechnicalErrors.
			WithLabelValues("decode_error", "parse_form", "critical").
			Inc()

		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	userData := models.UserSubData{
		Email:     r.FormValue("email"),
		City:      r.FormValue("city"),
		Frequency: r.FormValue("frequency"),
	}

	if userData.Email == "" || userData.City == "" || userData.Frequency == "" {
		h.logger.Warn().
			Interface("data", userData).
			Msg("missing required subscription fields")

		h.m.BusinessErrors.
			WithLabelValues("validation_error", "missing_fields", "warning").
			Inc()

		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	h.logger.Info().
		Str("email", userData.Email).
		Str("city", userData.City).
		Str("frequency", userData.Frequency).
		Msg("received subscribe request")

	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	payload, err := json.Marshal(userData)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to marshal subscription payload")

		h.m.TechnicalErrors.
			WithLabelValues("marshal_error", "json_marshal", "critical").
			Inc()

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		h.subURL+"/subscribe",
		bytes.NewReader(payload))
	if err != nil {
		h.logger.Error().
			Str("url", h.subURL+"/subscribe").
			Err(err).
			Msg("failed to create backend request")

		h.m.TechnicalErrors.
			WithLabelValues("request_error", "new_request", "critical").
			Inc()

		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("error sending subscribe request to backend")

		h.m.TechnicalErrors.
			WithLabelValues("network_error", "backend_unreachable", "critical").
			Inc()

		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			h.logger.Error().
				Err(err).
				Str("email", userData.Email).
				Str("city", userData.City).
				Msg("error closing response body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		h.logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("backend subscription error")

		// classify conflict vs other errors
		if resp.StatusCode == http.StatusConflict {
			h.m.BusinessErrors.
				WithLabelValues("subscription_exists", "409", "warning").
				Inc()
			http.Error(w, ErrSubscriptionExists.Error(), http.StatusConflict)
		} else {
			h.m.TechnicalErrors.
				WithLabelValues("backend_error", http.StatusText(resp.StatusCode), "critical").
				Inc()
			http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
		}
		return
	}

	// SUCCESS
	w.WriteHeader(http.StatusOK)
	n, err := w.Write([]byte(`{"message":"Subscribed successfully"}`))
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to write subscribe response")
		return
	}

	// record business metrics
	h.m.SubscriptionsCreated.
		WithLabelValues(userData.Frequency, "http_form").
		Inc()
	h.m.SubscriptionsActive.Inc()

	h.logger.Info().
		Int("bytes_written", n).
		Dur("duration_ms", time.Since(start)).
		Msg("subscription success")
}

// HandleConfirm
// @Summary Confirm subscription
// @Description Confirms the subscription using the token sent in email.
// @Tags subscription
// @Param token path string true "Confirmation token"
// @Success 200
// @Failure 400
// @Failure 404
// @Router /confirm/{token} [get]
func (h *Handler) HandleConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractTokenFromPath(r.URL.Path, "/api/v1/http/subscriptions/confirm/")
	if token == "" {
		h.logger.Warn().
			Str("path", r.URL.Path).
			Msg("confirm token not provided")
		http.Error(w, "Token not provided", http.StatusBadRequest)
		return
	}

	h.logger.Info().
		Str("token", token).
		Msg("confirming subscription")

	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.subURL+"/confirm/"+token, nil)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to create confirm request")
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("error sending confirm request to backend")
		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			h.logger.Error().
				Err(err).
				Str("token", token).
				Msg("error closing response body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			h.logger.Error().
				Err(err).
				Int("status", resp.StatusCode).
				Msg("failed to read response body")
			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}
		h.logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("backend confirm error")
		http.Error(w, "Failed to confirm subscription", resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	n, err := w.Write([]byte(`{"message":"Confirmed successfully"}`))
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to write confirm response")
		return
	}
	h.logger.Info().
		Int("bytes_written", n).
		Msg("confirm success")
}

// HandleUnsubscribe
// @Summary Unsubscribe
// @Description Unsubscribe from weather updates using the token.
// @Tags subscription
// @Param token path string true "Unsubscribe token"
// @Success 200
// @Failure 400
// @Failure 404
// @Router /unsubscribe/{token} [get]
func (h *Handler) HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodGet {
		h.logger.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("method not allowed")

		h.m.BusinessErrors.
			WithLabelValues("method_not_allowed", "405", "warning").
			Inc()

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractTokenFromPath(r.URL.Path, "/api/v1/http/subscriptions/unsubscribe/")
	if token == "" {
		h.logger.Warn().
			Str("path", r.URL.Path).
			Msg("unsubscribe token not provided")

		h.m.BusinessErrors.
			WithLabelValues("validation_error", "missing_token", "warning").
			Inc()

		http.Error(w, "Token not provided", http.StatusBadRequest)
		return
	}

	h.logger.Info().
		Str("token", token).
		Msg("unsubscribing token")

	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.subURL+"/unsubscribe/"+token, nil)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to create unsubscribe request")

		h.m.TechnicalErrors.
			WithLabelValues("request_error", "new_request", "critical").
			Inc()

		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("error sending unsubscribe request to backend")

		h.m.TechnicalErrors.
			WithLabelValues("network_error", "backend_unreachable", "critical").
			Inc()

		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		h.logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("backend unsubscribe error")

		h.m.TechnicalErrors.
			WithLabelValues("backend_error", http.StatusText(resp.StatusCode), "critical").
			Inc()

		http.Error(w, "Failed to unsubscribe", resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	n, err := w.Write([]byte(`{"message":"Unsubscribed successfully"}`))
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to write unsubscribe response")
		return
	}

	h.m.SubscriptionsActive.Dec()

	h.logger.Info().
		Int("bytes_written", n).
		Dur("duration_ms", time.Since(start)).
		Msg("unsubscribe success")
}

// extractTokenFromPath trims the prefix from the path and returns the token
func extractTokenFromPath(path, prefix string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	return path[len(prefix):]
}
