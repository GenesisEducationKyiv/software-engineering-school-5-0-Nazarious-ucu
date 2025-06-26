package email_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/services/email"

	"github.com/stretchr/testify/assert"
)

type mockEmailer struct {
	sendErr error
	body    string
}

func (m *mockEmailer) Send(to, subject, headers, body string) error {
	m.body = body
	return m.sendErr
}

func setupTemplate(t *testing.T) func() {
	t.Helper()
	dir := filepath.Join("internal", "templates")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("cannot create template dir: %v", err)
	}
	tmpl := filepath.Join(dir, "confirm_email.html")
	content := `<p>Hello {{.Email}}, please <a href="{{.Link}}">confirm</a></p>`
	if err := os.WriteFile(tmpl, []byte(content), 0o600); err != nil {
		t.Fatalf("cannot write template: %v", err)
	}
	return func() {
		err := os.RemoveAll("internal")
		if err != nil {
			return
		}
	}
}

func TestEmailService_SendConfirmation(t *testing.T) {
	teardown := setupTemplate(t)
	defer teardown()

	cases := []struct {
		name    string
		sendErr error
		wantErr bool
	}{
		{"success", nil, false},
		{"mailer error", errors.New("send failed"), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockEmailer{sendErr: tc.sendErr}
			svc := email.NewService(mock, "internal/templates")

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
		name    string
		sendErr error
		wantErr bool
	}{
		{"success", nil, false},
		{"mailer error", errors.New("smtp down"), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockEmailer{sendErr: tc.sendErr}
			svc := email.NewService(mock, "internal/templates")

			forecast := models.WeatherData{
				City:        "Kyiv",
				Temperature: 5.0,
				Condition:   "Snow",
			}
			err := svc.SendWeather("foo@bar.com", forecast.City, forecast)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				assert.Contains(t, mock.body, "Kyiv")
				assert.Contains(t, mock.body, "5.0°C")
				assert.Contains(t, mock.body, "Snow")
			}
		})
	}
}

func (m *mockEmailer) Body() string {
	return ""
}
