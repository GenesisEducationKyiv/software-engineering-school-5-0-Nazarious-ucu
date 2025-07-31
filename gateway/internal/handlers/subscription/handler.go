package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/metrics"
	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/models"
)

const (
	timeoutDuration = 10 * time.Second
)

// var ErrSubscriptionExists = errors.New("subscription already exists")

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
	m *metrics.Metrics,
) *Handler {
	return &Handler{
		client: client,
		subURL: subscribeURL,
		logger: logger,
		m:      m,
	}
}

func (h *Handler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	clientIP := r.RemoteAddr

	h.logger.Debug().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", clientIP).
		Msg("start HandleSubscribe")

	if r.Method != http.MethodPost {
		h.logger.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
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
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("failed to parse form")

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
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Bool("email_empty", userData.Email == "").
			Bool("city_empty", userData.City == "").
			Bool("frequency_empty", userData.Frequency == "").
			Dur("duration_ms", time.Since(start)).
			Msg("missing required subscription fields")

		h.m.BusinessErrors.
			WithLabelValues("validation_error", "missing_fields", "warning").
			Inc()

		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	h.logger.Info().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", clientIP).
		Str("city", userData.City).
		Str("frequency", userData.Frequency).
		Msg("received subscribe request")

	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	payload, err := json.Marshal(userData)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("failed to marshal subscription payload")

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
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Str("url", h.subURL+"/subscribe").
			Dur("duration_ms", time.Since(start)).
			Msg("failed to create backend request")

		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("error sending subscribe request to backend")

		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			h.logger.Error().
				Err(err).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("city", userData.City).
				Str("client_ip", clientIP).
				Msg("error closing response body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			h.logger.Error().
				Err(err).
				Int("status", resp.StatusCode).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("client_ip", clientIP).
				Dur("duration_ms", time.Since(start)).
				Msg("failed to read response body")
			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}
		h.logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("backend subscription error")
		return
	}

	// SUCCESS
	w.WriteHeader(http.StatusOK)
	n, err := w.Write([]byte(`{"message":"Subscribed successfully"}`))
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("failed to write subscribe response")
		return
	}

	// record business metrics
	h.m.SubscriptionsCreated.
		WithLabelValues(userData.Frequency, "http_form").
		Inc()

	h.logger.Info().
		Int("bytes_written", n).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", clientIP).
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
	start := time.Now()
	clientIP := r.RemoteAddr

	// Entry log
	h.logger.Debug().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", clientIP).
		Msg("start HandleConfirm")

	// Method check
	if r.Method != http.MethodGet {
		h.logger.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Msg("method not allowed")

		h.m.BusinessErrors.
			WithLabelValues("method_not_allowed", "405", "warning").
			Inc()

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token
	token := extractTokenFromPath(r.URL.Path, "/api/v1/http/subscriptions/confirm/")
	if token == "" {
		h.logger.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Msg("confirm token not provided")

		h.m.BusinessErrors.
			WithLabelValues("validation_error", "missing_token", "warning").
			Inc()

		http.Error(w, "Token not provided", http.StatusBadRequest)
		return
	}

	h.logger.Info().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", clientIP).
		Str("token", token).
		Msg("confirming subscription")

	// Forward to backend
	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.subURL+"/confirm/"+token, nil)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("failed to create confirm request")

		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("error sending confirm request to backend")

		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			h.logger.Error().
				Err(err).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("client_ip", clientIP).
				Str("token", token).
				Msg("error closing response body")
		}
	}(resp.Body)

	// Handle non-OK response
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			h.logger.Error().
				Err(readErr).
				Str("body", string(body)).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("client_ip", clientIP).
				Str("client_ip", clientIP).
				Msg("failed to read response body")

			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}

		h.logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("backend confirm error")

		http.Error(w, "Failed to confirm subscription", resp.StatusCode)
		return
	}

	// Success
	w.WriteHeader(http.StatusOK)
	n, err := w.Write([]byte(`{"message":"Confirmed successfully"}`))
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("failed to write confirm response")
		return
	}

	h.m.SubscriptionsActive.Inc()

	h.logger.Info().
		Int("bytes_written", n).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", clientIP).
		Dur("duration_ms", time.Since(start)).
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
	clientIP := r.RemoteAddr

	// Entry log
	h.logger.Debug().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", clientIP).
		Msg("start HandleUnsubscribe")

	// Method check
	if r.Method != http.MethodGet {
		h.logger.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Msg("method not allowed")

		h.m.BusinessErrors.
			WithLabelValues("method_not_allowed", "405", "warning").
			Inc()

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token
	token := extractTokenFromPath(r.URL.Path, "/api/v1/http/subscriptions/unsubscribe/")
	if token == "" {
		h.logger.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Msg("unsubscribe token not provided")

		h.m.BusinessErrors.
			WithLabelValues("validation_error", "missing_token", "warning").
			Inc()

		http.Error(w, "Token not provided", http.StatusBadRequest)
		return
	}

	h.logger.Info().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", clientIP).
		Str("token", token).
		Msg("unsubscribing token")

	// Forward to backend
	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.subURL+"/unsubscribe/"+token, nil)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("failed to create unsubscribe request")

		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("error sending unsubscribe request to backend")

		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			h.logger.Error().
				Err(err).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("client_ip", clientIP).
				Str("token", token).
				Msg("error closing response body")
		}
	}(resp.Body)

	// Handle non-OK response
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			h.logger.Error().
				Err(readErr).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("client_ip", clientIP).
				Msg("failed to read response body")

			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}

		h.logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("backend unsubscribe error")

		http.Error(w, "Failed to unsubscribe", resp.StatusCode)
		return
	}

	// Success
	w.WriteHeader(http.StatusOK)
	n, err := w.Write([]byte(`{"message":"Unsubscribed successfully"}`))
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("failed to write unsubscribe response")
		return
	}

	h.m.SubscriptionsActive.Dec()

	h.logger.Info().
		Int("bytes_written", n).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", clientIP).
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
