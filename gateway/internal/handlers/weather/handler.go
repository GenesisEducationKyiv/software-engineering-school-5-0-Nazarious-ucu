package weather

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"

	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/metrics"
)

const (
	timeoutDuration = 10 * time.Second
)

// Handler proxies weather requests and records logs/metrics.
type Handler struct {
	client  *http.Client
	baseURL string
	logger  zerolog.Logger
	m       *metrics.Metrics
}

// NewHandler creates a new weather HTTP handler.
func NewHandler(
	client *http.Client,
	weatherServiceBaseURL string,
	logger zerolog.Logger,
	m *metrics.Metrics,
) *Handler {
	return &Handler{client: client, baseURL: weatherServiceBaseURL, logger: logger, m: m}
}

// HandleGetWeather handles GET /api/v1/http/weather?city={city}.
func (h *Handler) HandleGetWeather(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	clientIP := r.RemoteAddr

	// Entry log
	h.logger.Debug().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", clientIP).
		Msg("start HandleGetWeather")

	// Method check
	if r.Method != http.MethodGet {
		h.logger.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Msg("method not allowed")

		// Only domain metrics here
		h.m.WeatherFailures.
			WithLabelValues(r.Method, r.URL.Path, "5xx").
			Inc()

		h.logger.Info().
			Dur("duration_ms", time.Since(start)).
			Msg("method failed: not allowed")

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Query param check
	city := r.URL.Query().Get("city")
	if city == "" {
		h.logger.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Msg("missing city query parameter")

		// domain failure metric
		h.m.WeatherFailures.
			WithLabelValues(r.Method, r.URL.Path, "4xx").
			Inc()

		h.logger.Info().
			Dur("duration_ms", time.Since(start)).
			Msg("method failed: missing city")

		http.Error(w, "city query parameter is required", http.StatusBadRequest)
		return
	}

	h.logger.Info().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", clientIP).
		Str("city", city).
		Msg("forwarding weather request")

	// Forward to backend
	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	targetURL, err := url.Parse(h.baseURL + "/weather")
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Msg("failed to parse weather service URL")

		// domain failure metric
		h.m.WeatherFailures.
			WithLabelValues(r.Method, r.URL.Path, "5xx").
			Inc()

		h.logger.Info().
			Dur("duration_ms", time.Since(start)).
			Msg("parsing URL failed")

		http.Error(w, "Failed to parse weather service URL", http.StatusInternalServerError)
		return
	}
	targetURL.RawQuery = url.Values{"city": {city}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Msg("failed to create proxied request")

		// domain failure metric
		h.m.WeatherFailures.
			WithLabelValues(r.Method, r.URL.Path, "5xx").
			Inc()

		h.logger.Info().
			Dur("duration_ms", time.Since(start)).
			Msg("creating request failed")

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
			Msg("error forwarding request to backend")

		// domain failure metric
		h.m.WeatherFailures.
			WithLabelValues(r.Method, r.URL.Path, "5xx").
			Inc()

		h.logger.Info().
			Dur("duration_ms", time.Since(start)).
			Msg("forwarding request failed")

		http.Error(w, "Failed to contact weather service", http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("client_ip", clientIP).
				Msg("error closing response body")
		}
	}(resp.Body)

	// Proxy response
	w.WriteHeader(resp.StatusCode)

	written, err := io.Copy(w, resp.Body)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Msg("error writing response body")
		return
	}

	// Record domain metrics
	h.m.WeatherProcessingTime.WithLabelValues(
		r.Method,
		r.URL.Path,
		h.m.GetStatusClass(resp.StatusCode)).Observe(time.Since(start).Seconds())

	if resp.StatusCode >= http.StatusInternalServerError {
		h.m.WeatherFailures.WithLabelValues(r.Method, r.URL.Path, h.m.GetStatusClass(resp.StatusCode)).Inc()
	}

	// Final log
	if written == 0 {
		h.logger.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("no data written to response")
	} else {
		h.logger.Info().
			Int("status", resp.StatusCode).
			Int("bytes", int(written)).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", clientIP).
			Dur("duration_ms", time.Since(start)).
			Msg("successfully proxied weather response")
	}
}
