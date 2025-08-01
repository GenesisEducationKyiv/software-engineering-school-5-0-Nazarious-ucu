//go:build unit

package notifier_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/pkg/logger"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/metrics"
	"github.com/stretchr/testify/require"

	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/notifier"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const freqTest = "hourly"

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) GetConfirmedByFrequency(
	ctx context.Context,
	frequency string,
) ([]models.Subscription, error) {
	args := m.Called(frequency, ctx)
	data, ok := args.Get(0).([]models.Subscription)
	if !ok {
		return []models.Subscription{}, nil
	}
	return data, args.Error(1)
}

func (m *mockRepo) UpdateLastSent(ctx context.Context, subscriptionID int) error {
	args := m.Called(ctx, subscriptionID)
	return args.Error(0)
}

type mockWeather struct {
	mock.Mock
}

func (m *mockWeather) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	args := m.Called(ctx, city)
	data, ok := args.Get(0).(models.WeatherData)
	if !ok {
		return models.WeatherData{}, args.Error(1)
	}

	return data, args.Error(1)
}

type mockEmail struct {
	mock.Mock
}

func (m *mockEmail) SendWeather(ctx context.Context, to string, forecast models.WeatherData) error {
	args := m.Called(ctx, to, forecast)
	return args.Error(0)
}

func Test_sendOne_Success(t *testing.T) {
	const (
		city  = "Kyiv"
		email = "user@kyiv.ua"
	)
	sub := models.Subscription{ID: 1, City: city, Email: email}

	mockR := &mockRepo{}
	mockW := &mockWeather{}
	mockE := &mockEmail{}

	mockR.On("UpdateLastSent", mock.Anything, sub.ID).Return(nil)

	forecast := models.WeatherData{City: city, Temperature: 5.0, Condition: "Sunny"}
	mockW.On("GetByCity", mock.Anything, city).Return(forecast, nil)
	mockE.On("SendWeather", mock.Anything, email, forecast).Return(nil)

	t.Cleanup(func() {
		mockR.AssertExpectations(t)
		mockW.AssertExpectations(t)
		mockE.AssertExpectations(t)
	})

	l, err := logger.NewLogger("logs/subscriptions_test.log", "notifier_test")
	require.NoError(t, err)

	m := metrics.NewMetrics("notifier_test", &sql.DB{}, "test")

	n := notifier.New(mockR, mockW, mockE, l, "@every 1h", "0 0 9 * * *", m)
	err = n.SendOne(context.Background(), sub)

	assert.NoError(t, err)
}

func Test_sendOne_Error_APIError(t *testing.T) {
	const city = "Lviv"
	sub := models.Subscription{ID: 2, City: city, Email: "x"}

	// case1: GetByCity error
	rm := &mockRepo{}
	wm := &mockWeather{}
	em := &mockEmail{}

	wm.On("GetByCity", mock.Anything, city).Return(models.WeatherData{}, errors.New("api down"))

	t.Cleanup(func() {
		rm.AssertExpectations(t)
		wm.AssertExpectations(t)
		em.AssertExpectations(t)
	})

	l, err := logger.NewLogger("logs/subscriptions_test.log", "notifier_test")
	require.NoError(t, err)

	m := metrics.NewMetrics("notifier_test", &sql.DB{}, "test")

	n1 := notifier.New(rm, wm, em, l, "@every 1h", "0 0 9 * * *", m)
	err1 := n1.SendOne(context.Background(), sub)
	assert.Error(t, err1)
}

func Test_sendOne_Error_EmailError(t *testing.T) {
	const city = "Lviv"
	sub := models.Subscription{ID: 2, City: city, Email: "x"}

	rm := &mockRepo{}
	wm := &mockWeather{}

	em := &mockEmail{}

	wm.On("GetByCity", mock.Anything, city).Return(models.WeatherData{City: city}, nil)
	em.On("SendWeather", mock.Anything, sub.Email, mock.Anything).Return(errors.New("smtp fail"))

	t.Cleanup(func() {
		rm.AssertExpectations(t)
		wm.AssertExpectations(t)
		em.AssertExpectations(t)
	})

	l, err := logger.NewLogger("logs/subscriptions_test.log", "notifier_test")
	require.NoError(t, err)

	m := metrics.NewMetrics("notifier_test", &sql.DB{}, "test")

	// UpdateLastSent should not be called on send fail
	n2 := notifier.New(rm, wm, em, l, "@every 1h", "0 0 9 * * *", m)
	err2 := n2.SendOne(context.Background(), sub)
	assert.Error(t, err2)
}

func Test_runDue_Success(t *testing.T) {
	city1, city2 := "Odesa", "Kharkiv"
	subs := []models.Subscription{{ID: 10, City: city1, Email: "a"}, {ID: 20, City: city2, Email: "b"}}

	rm := &mockRepo{}
	wm := &mockWeather{}
	em := &mockEmail{}

	rm.On("GetConfirmedByFrequency", freqTest, mock.Anything).Return(subs, nil)
	rm.On("UpdateLastSent", mock.Anything, 10).Return(nil)
	rm.On("UpdateLastSent", mock.Anything, 20).Return(nil)

	// weather calls
	wm.On("GetByCity", mock.Anything, city1).Return(models.WeatherData{City: city1}, nil)
	wm.On("GetByCity", mock.Anything, city2).Return(models.WeatherData{City: city2}, nil)

	// email sends
	em.On("SendWeather", mock.Anything, "a", mock.Anything).Return(nil)
	em.On("SendWeather", mock.Anything, "b", mock.Anything).Return(nil)

	t.Cleanup(func() {
		rm.AssertExpectations(t)
		wm.AssertExpectations(t)
		em.AssertExpectations(t)
	})

	l, err := logger.NewLogger("logs/subscriptions_test.log", "notifier_test")
	require.NoError(t, err)

	m := metrics.NewMetrics("notifier_test", &sql.DB{}, "test")

	n := notifier.New(rm, wm, em, l, "@every 1h", "0 0 9 * * *", m)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	n.RunDue(ctx, freqTest)
}

func Test_runDue_FetchError(t *testing.T) {
	mockR := &mockRepo{}
	mockE := &mockEmail{}
	mockW := &mockWeather{}

	l, err := logger.NewLogger("logs/subscriptions_test.log", "notifier_test")
	require.NoError(t, err)

	m := metrics.NewMetrics("notifier_test", &sql.DB{}, "test")

	n := notifier.New(mockR, mockW, mockE, l, "@every 1h", "0 0 9 * * *", m)

	mockR.On("GetConfirmedByFrequency", freqTest, mock.Anything).
		Return([]models.Subscription{}, errors.New("db down"))

	mockR.AssertNumberOfCalls(t, "UpdateLastSent", 0)
	mockW.AssertNumberOfCalls(t, "GetByCity", 0)
	mockE.AssertNumberOfCalls(t, "SendWeather", 0)

	n.RunDue(context.Background(), freqTest)

	t.Cleanup(func() {
		mockR.AssertExpectations(t)
	})
}
