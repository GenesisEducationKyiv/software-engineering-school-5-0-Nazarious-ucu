//go:build unit

package notifier_test

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/notifier"
)

const freqTest = "hourly"

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) GetConfirmedByFrequency(
	frequency string,
	ctx context.Context,
) ([]models.Subscription, error) {
	args := m.Called(frequency, ctx)
	data, ok := args.Get(0).([]models.Subscription)
	if !ok {
		return []models.Subscription{}, nil
	}
	return data, args.Error(1)
}

func (m *mockRepo) UpdateLastSent(subscriptionID int, ctx context.Context) error {
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

func (m *mockEmail) SendWeather(to string, forecast models.WeatherData) error {
	args := m.Called(to, forecast)
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
	mockE.On("SendWeather", email, forecast).Return(nil)

	t.Cleanup(func() {
		mockR.AssertExpectations(t)
		mockW.AssertExpectations(t)
		mockE.AssertExpectations(t)
	})

	n := notifier.New(mockR, mockW, mockE, log.New(io.Discard, "", 0), "@every 1h", "0 0 9 * * *")
	err := n.SendOne(context.Background(), sub)

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

	n1 := notifier.New(rm, wm, em, log.New(io.Discard, "", 0), "@every 1h", "0 0 9 * * *")
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
	em.On("SendWeather", sub.Email, mock.Anything).Return(errors.New("smtp fail"))

	t.Cleanup(func() {
		rm.AssertExpectations(t)
		wm.AssertExpectations(t)
		em.AssertExpectations(t)
	})

	// UpdateLastSent should not be called on send fail
	n2 := notifier.New(rm, wm, em, log.New(io.Discard, "", 0), "@every 1h", "0 0 9 * * *")
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
	em.On("SendWeather", "a", mock.Anything).Return(nil)
	em.On("SendWeather", "b", mock.Anything).Return(nil)

	t.Cleanup(func() {
		rm.AssertExpectations(t)
		wm.AssertExpectations(t)
		em.AssertExpectations(t)
	})

	n := notifier.New(rm, wm, em, log.New(io.Discard, "", 0), "@every 1h", "0 0 9 * * *")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	n.RunDue(ctx, freqTest)
}

func Test_runDue_FetchError(t *testing.T) {
	mockR := &mockRepo{}
	mockE := &mockEmail{}
	mockW := &mockWeather{}

	n := notifier.New(mockR, mockW, mockE, log.New(io.Discard, "", 0), "@every 1h", "0 0 9 * * *")

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
