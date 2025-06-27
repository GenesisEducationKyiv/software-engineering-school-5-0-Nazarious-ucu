package email_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/email"

	"github.com/stretchr/testify/assert"
)

type mockEmailer struct {
	mock.Mock
}

func (m *mockEmailer) Send(to, subject, headers, body string) error {
	args := m.Called(to, subject, headers, body)
	return args.Error(0)
}

func TestEmailService_SendConfirmation(t *testing.T) {
	cases := []struct {
		name    string
		sendErr error
		wantErr bool
	}{
		{"success", nil, false},
		{"mailer error", errors.New("send failed"), true},
	}

	m := &mockEmailer{}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m.On("Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.sendErr).Once()
			t.Cleanup(func() {
				m.AssertExpectations(t)
			})

			svc := email.NewService(m, "../../templates")
			err := svc.SendConfirmation("user@example.com", "TOKEN123")
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailService_SendWeather(t *testing.T) {
	cases := []struct {
		name         string
		sendErr      error
		forecastSend models.WeatherData
	}{
		{
			"success", nil,
			models.WeatherData{City: "Kyiv", Temperature: 5.0, Condition: "Snow"},
		},
		{
			"mailer error", errors.New("smtp down"),
			models.WeatherData{City: "", Temperature: 0, Condition: ""},
		},
	}
	m := &mockEmailer{}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m.On("Send",
				mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(arg interface{}) bool {
					s, ok := arg.(string)
					if !ok {
						return false
					}
					return strings.Contains(s, tc.forecastSend.City) &&
						strings.Contains(s, tc.forecastSend.Condition) &&
						strings.Contains(s, strconv.FormatFloat(tc.forecastSend.Temperature, 'f', 1, 64))
				})).Return(tc.sendErr).Once()

			t.Cleanup(func() {
				m.AssertExpectations(t)
			})

			svc := email.NewService(m, "internal/templates")

			err := svc.SendWeather("foo@bar.com", tc.forecastSend)

			assert.Equal(t, tc.sendErr, err)
		})
	}
}

func (m *mockEmailer) Body() string {
	return ""
}
