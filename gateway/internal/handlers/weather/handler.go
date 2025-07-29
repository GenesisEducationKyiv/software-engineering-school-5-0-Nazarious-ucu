package weather

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

const timeoutDuration = 10 * time.Second

type Handler struct {
	client  *http.Client
	baseURL string
	logger  *zap.SugaredLogger
}

func NewHandler(client *http.Client, weatherServiceBaseURL string, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		client:  client,
		baseURL: weatherServiceBaseURL,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/http/weather", h.handleGetWeather)
}

func (h *Handler) handleGetWeather(w http.ResponseWriter, r *http.Request) {
	city := r.URL.Query().Get("city")
	if city == "" {
		h.logger.Warn("missing city query parameter")
		http.Error(w, "city query parameter is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	targetURL, err := url.Parse(h.baseURL + "/weather")
	if err != nil {
		h.logger.Errorw("failed to parse weather service URL",
			"baseURL", h.baseURL, "error", err,
		)
		http.Error(w, "Failed to parse weather service URL", http.StatusInternalServerError)
		return
	}
	q := targetURL.Query()
	q.Set("city", city)
	targetURL.RawQuery = q.Encode()

	// Create proxied request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		h.logger.Errorw("failed to create proxied request",
			"url", targetURL.String(), "error", err,
		)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Errorw("error forwarding request",
			"url", targetURL.String(), "error", err,
		)
		http.Error(w, "Failed to contact weather service", http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			h.logger.Errorw("error closing response body",
				"url", targetURL.String(), "error", err,
			)
		}
	}(resp.Body)

	w.WriteHeader(resp.StatusCode)

	written, err := io.Copy(w, resp.Body)
	if err != nil {
		h.logger.Errorw("error writing response body", "error", err)
		return
	}
	if written == 0 {
		h.logger.Warnw("no data written to response", "url", targetURL.String())
	} else {
		h.logger.Infow("successfully proxied weather response",
			"city", city, "status", resp.StatusCode, "bytes", written,
		)
	}
}
