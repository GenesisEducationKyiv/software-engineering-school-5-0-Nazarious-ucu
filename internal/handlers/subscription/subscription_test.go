//go:build unit

package subscription_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/subscription"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) Subscribe(ctx context.Context, data subscription.UserSubData) error {
	args := m.Called(ctx, data)

	return args.Error(0)
}

func (m *mockService) Confirm(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

func (m *mockService) Unsubscribe(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)

	return args.Bool(0), args.Error(1)
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
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)

			m := &mockService{}
			m.On("Subscribe", mock.Anything, mock.Anything).Return(tc.mockErr).Maybe()

			t.Cleanup(func() {
				m.AssertExpectations(t)
			})

			req, err := http.NewRequest(http.MethodPost, "/subscribe",
				strings.NewReader(tc.body))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			h := subscription.NewHandler(m)
			h.Subscribe(c)

			assert.Equal(t, tc.wantCode, rec.Code)
			assert.JSONEq(t, tc.wantBody, rec.Body.String())
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
			mockOK:   true,
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
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)

			m := &mockService{}

			m.On("Confirm", mock.Anything, mock.Anything).Return(tc.mockOK, tc.mockErr).Once()

			t.Cleanup(func() {
				m.AssertExpectations(t)
			})

			req, err := http.NewRequest(http.MethodGet, "/confirm/"+tc.token, nil)

			require.NoError(t, err)

			c.Request = req

			h := subscription.NewHandler(m)
			h.Confirm(c)

			assert.Equal(t, tc.wantCode, rec.Code)
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
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)

			m := &mockService{}

			m.On("Unsubscribe", mock.Anything, mock.Anything).Return(tc.mockOK, tc.mockErr).Once()

			t.Cleanup(func() {
				m.AssertExpectations(t)
			})

			req := httptest.NewRequest(http.MethodGet, "/unsubscribe/"+tc.token, nil)

			c.Request = req

			h := subscription.NewHandler(m)
			h.Unsubscribe(c)

			assert.Equal(t, tc.wantCode, rec.Code)
		})
	}
}
