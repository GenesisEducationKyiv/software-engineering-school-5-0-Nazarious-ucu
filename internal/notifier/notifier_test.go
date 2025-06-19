package notifier_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/notifier"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/repository"
	service "github.com/Nazarious-ucu/weather-subscription-api/internal/services"
	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	subs           []repository.Subscription
	getErr, updErr error
	updatedIDs     []int
}

func (m *mockRepo) GetConfirmed() ([]repository.Subscription, error) {
	return m.subs, m.getErr
}

func (m *mockRepo) UpdateLastSent(subscriptionID int) error {
	m.updatedIDs = append(m.updatedIDs, subscriptionID)
	return m.updErr
}

type mockWeatherSvc struct {
	data       service.WeatherData
	err        error
	calledWith []string
}

func (m *mockWeatherSvc) GetByCity(ctx context.Context, city string) (service.WeatherData, error) {
	m.calledWith = append(m.calledWith, city)
	return m.data, m.err
}

type mockEmailSender struct {
	err          error
	sentTo       []string
	sentCity     []string
	sentForecast []service.WeatherData
}

func (m *mockEmailSender) SendWeather(to, city string, forecast service.WeatherData) error {
	m.sentTo = append(m.sentTo, to)
	m.sentCity = append(m.sentCity, city)
	m.sentForecast = append(m.sentForecast, forecast)
	return m.err
}

func TestShouldSendUpdate(t *testing.T) {
	now := time.Date(2025, 6, 18, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		sub      repository.Subscription
		wantSend bool
	}{
		{"no last sent", repository.Subscription{LastSentAt: nil, Frequency: "hourly"}, true},
		{"hourly - just sent", repository.Subscription{LastSentAt: ptrTime(now), Frequency: "hourly"}, false},
		{"hourly - overdue", repository.Subscription{LastSentAt: ptrTime(now.Add(-2 * time.Hour)),
			Frequency: "hourly"}, true},
		{"daily - just sent", repository.Subscription{LastSentAt: ptrTime(now), Frequency: "daily"}, false},
		{"daily - overdue", repository.Subscription{LastSentAt: ptrTime(now.Add(-25 * time.Hour)),
			Frequency: "daily"}, true},
		{"unknown freq", repository.Subscription{LastSentAt: ptrTime(now.Add(-100 * time.Hour)),
			Frequency: "weekly"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n := notifier.Notifier{}
			got := n.ShouldSendUpdate(tc.sub, now)
			assert.Equal(t, tc.wantSend, got)
		})
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestSendWeatherUpdate(t *testing.T) {
	const city = "Kyiv"
	const emailAddr = "user@kyiv.ua"

	baseSub := repository.Subscription{ID: 42, Email: emailAddr, City: city}

	t.Run("success path", func(t *testing.T) {
		mockRepo := &mockRepo{}
		mockW := &mockWeatherSvc{
			data: service.WeatherData{City: city, Temperature: 10.0, Condition: "Clear"},
		}
		mockEmail := &mockEmailSender{}

		n := notifier.Notifier{
			Repo:           mockRepo,
			WeatherService: mockW,
			EmailService:   mockEmail,
		}

		err := n.SendWeatherUpdate(baseSub)
		assert.NoError(t, err)

		assert.Equal(t, []string{city}, mockW.calledWith)

		assert.Equal(t, []string{emailAddr}, mockEmail.sentTo)
		assert.Equal(t, []string{city}, mockEmail.sentCity)
		assert.Equal(t, []service.WeatherData{{City: city, Temperature: 10.0, Condition: "Clear"}},
			mockEmail.sentForecast)

		assert.Equal(t, []int{42}, mockRepo.updatedIDs)
	})

	t.Run("weather fetch error", func(t *testing.T) {
		mockRepo := &mockRepo{}
		mockW := &mockWeatherSvc{err: errors.New("api down")}
		mockEmail := &mockEmailSender{}

		n := notifier.Notifier{
			Repo:           mockRepo,
			WeatherService: mockW,
			EmailService:   mockEmail,
		}

		err := n.SendWeatherUpdate(baseSub)
		assert.Error(t, err, "should return weather error")

		assert.Empty(t, mockEmail.sentTo)
		assert.Empty(t, mockRepo.updatedIDs)
	})

	t.Run("email send error", func(t *testing.T) {
		mockRepo := &mockRepo{}
		mockW := &mockWeatherSvc{data: service.WeatherData{City: city}}
		mockEmail := &mockEmailSender{err: errors.New("smtp not available")}

		n := notifier.Notifier{
			Repo:           mockRepo,
			WeatherService: mockW,
			EmailService:   mockEmail,
		}

		err := n.SendWeatherUpdate(baseSub)
		assert.Error(t, err, "should return email error")
		assert.Empty(t, mockRepo.updatedIDs)
	})
}
