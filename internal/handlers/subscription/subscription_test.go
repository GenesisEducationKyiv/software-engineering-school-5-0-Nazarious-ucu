package subscription_test

import (
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/subscription"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
)

type mockService struct {
	subErr     error
	confirmOK  bool
	confirmErr error
	unsubOK    bool
	unsubErr   error
}

func (m *mockService) Subscribe(data models.UserSubData) error {
	return m.subErr
}

func (m *mockService) Confirm(token string) (bool, error) {
	return m.confirmOK, m.confirmErr
}

func (m *mockService) Unsubscribe(token string) (bool, error) {
	return m.unsubOK, m.unsubErr
}

func setupRouter(svc *mockService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	h := subscription.NewHandler(svc)
	r.POST("/subscribe", h.Subscribe)
	r.GET("/confirm/:token", h.Confirm)
	r.GET("/unsubscribe/:token", h.Unsubscribe)

	return r
}

func TestSubscribeEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		mockErr  error
		wantCode int
		wantBody string
	}{
		{
			name:     "missing fields",
			body:     `{"email": "test@a.com", "city": "Kyiv"}`,
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"Missing required fields"}`,
		},
		{
			name:     "service error",
			body:     `{"email": "test@gmail.com", "city": "Lviv", "frequency": "hourly"}`,
			mockErr:  errors.New("fail"),
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"Internal server error"}`,
		},
		{
			name:     "success",
			body:     `{"email": "test@gmail.com", "city": "Lviv", "frequency": "hourly"}`,
			wantCode: http.StatusOK,
			wantBody: `{"message":"Subscribed successfully"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockService{subErr: tc.mockErr}
			router := setupRouter(mock)

			req := httptest.NewRequest(http.MethodPost, "/subscribe",
				strings.NewReader(tc.body))
			log.Println(strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.wantCode, w.Code)
			assert.JSONEq(t, tc.wantBody, w.Body.String())
		})
	}
}

func TestConfirmEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		token    string
		mockOK   bool
		mockErr  error
		wantCode int
	}{
		{
			name:     "service error",
			token:    "tok1",
			mockOK:   false,
			mockErr:  errors.New("fail"),
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "invalid token",
			token:    "tok2",
			mockOK:   false,
			mockErr:  nil,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "success",
			token:    "tok3",
			mockOK:   true,
			mockErr:  nil,
			wantCode: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockService{confirmOK: tc.mockOK, confirmErr: tc.mockErr}
			router := setupRouter(mock)

			req := httptest.NewRequest(http.MethodGet, "/confirm/"+tc.token, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.wantCode, w.Code)
		})
	}
}

func TestUnsubscribeEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		token    string
		mockOK   bool
		mockErr  error
		wantCode int
	}{
		{
			name:     "service error",
			token:    "tokA",
			mockOK:   false,
			mockErr:  errors.New("fail"),
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "invalid token",
			token:    "tokB",
			mockOK:   false,
			mockErr:  nil,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "success",
			token:    "tokC",
			mockOK:   true,
			mockErr:  nil,
			wantCode: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockService{unsubOK: tc.mockOK, unsubErr: tc.mockErr}
			router := setupRouter(mock)

			req := httptest.NewRequest(http.MethodGet, "/unsubscribe/"+tc.token, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.wantCode, w.Code)
		})
	}
}
