package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/gateway/internal/models"
)

const timeoutDuration = 10 * time.Second

var ErrSubscriptionExists = errors.New("subscription already exists")

type Handler struct {
	client *http.Client
	subURL string
	logger *log.Logger
}

func NewHandler(client *http.Client, subscribeURL string, logger *log.Logger) *Handler {
	return &Handler{client: client, subURL: subscribeURL, logger: logger}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/http/subscriptions/subscribe", h.handleSubscribe)
	mux.HandleFunc("/api/v1/http/subscriptions/confirm/", h.handleConfirm)
	mux.HandleFunc("/api/v1/http/subscriptions/unsubscribe/", h.handleUnsubscribe)
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
func (h *Handler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	userData := models.UserSubData{
		Email:     r.FormValue("email"),
		City:      r.FormValue("city"),
		Frequency: r.FormValue("frequency"),
	}

	if userData.Email == "" || userData.City == "" || userData.Frequency == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	h.logger.Printf("Received subscribe request: %+v", userData)

	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	jsonBytes, err := json.Marshal(userData)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		h.subURL+"/subscribe",
		bytes.NewReader(jsonBytes))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			h.logger.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			h.logger.Printf("Failed to read response body: %s", err)
			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}
		h.logger.Printf("Subscription error: %s", string(bodyBytes))

		switch resp.StatusCode {
		case http.StatusBadRequest:
			http.Error(w, "Invalid request", http.StatusBadRequest)
		case http.StatusConflict:
			http.Error(w, ErrSubscriptionExists.Error(), http.StatusConflict)
		default:
			http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	write, err := w.Write([]byte(`{"message":"Subscribed successfully"}`))
	if err != nil {
		h.logger.Printf("Failed to write response: %s", err)
		return
	}
	if write == 0 {
		h.logger.Println("No data written to response")
		return
	}
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
func (h *Handler) handleConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractTokenFromPath(r.URL.Path, "/api/v1/http/subscriptions/confirm/")
	if token == "" {
		http.Error(w, "Token not provided", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.subURL+"/confirm/"+token, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			h.logger.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			h.logger.Printf("Failed to read response body: %s", err)
			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}
		h.logger.Printf("Confirm error: %s", string(bodyBytes))
		http.Error(w, "Failed to confirm subscription", resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	write, err := w.Write([]byte(`{"message":"Confirmed successfully"}`))
	if err != nil {
		h.logger.Printf("Failed to write response: %s", err)
		return
	}
	if write == 0 {
		h.logger.Println("No data written to response")
		return
	}
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
func (h *Handler) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractTokenFromPath(r.URL.Path, "/api/v1/http/subscriptions/unsubscribe/")
	if token == "" {
		http.Error(w, "Token not provided", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.subURL+"/unsubscribe/"+token, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			h.logger.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to unsubscribe", resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	write, err := w.Write([]byte(`{"message":"Unsubscribed successfully"}`))
	if err != nil {
		h.logger.Printf("Failed to write response: %s", err)
		return
	}
	if write == 0 {
		h.logger.Println("No data written to response")
		return
	}
}

// extractTokenFromPath trims the prefix from the path and returns the token
func extractTokenFromPath(path, prefix string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	return path[len(prefix):]
}
