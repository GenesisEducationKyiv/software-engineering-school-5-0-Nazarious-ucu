package weather

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const timeoutDuration = 10 * time.Second

type Handler struct {
	client  *http.Client
	baseURL string
	logger  *log.Logger
}

func NewHandler(client *http.Client, weatherServiceBaseURL string, logger *log.Logger) *Handler {
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
		http.Error(w, "city query parameter is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	targetURL, err := url.Parse(h.baseURL + "/weather")
	if err != nil {
		http.Error(w, "Failed to parse weather service URL", http.StatusInternalServerError)
		return
	}
	query := targetURL.Query()
	query.Set("city", city)
	targetURL.RawQuery = query.Encode()

	// Create proxied request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Printf("Error forwarding request: %v", err)
		http.Error(w, "Failed to contact weather service", http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			h.logger.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	// Copy status and body
	w.WriteHeader(resp.StatusCode)
	written, err := io.Copy(w, resp.Body)
	if err != nil {
		h.logger.Printf("Error writing response: %v", err)
		return
	}
	if written == 0 {
		h.logger.Println("No data written to response")
		return
	}
}
