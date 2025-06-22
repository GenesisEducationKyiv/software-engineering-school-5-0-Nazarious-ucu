// internal/notifier/notifier_test.go
package notifier_test

import (
	"context"
	"errors"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/notifier"
)

const freqTest = "hourly"

type mockRepo struct {
	subs       []models.Subscription
	getErr     error
	updatedIDs []int
	mu         sync.Mutex
}

func (m *mockRepo) GetConfirmedByFrequency(
	frequency string,
	ctx context.Context,
) ([]models.Subscription, error) {
	return m.subs, m.getErr
}

func (m *mockRepo) UpdateLastSent(subscriptionID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updatedIDs = append(m.updatedIDs, subscriptionID)
	return nil
}

type mockWeather struct {
	data       models.WeatherData
	err        error
	calledWith []string
	mu         sync.Mutex
}

func (m *mockWeather) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	m.mu.Lock()
	m.calledWith = append(m.calledWith, city)
	m.mu.Unlock()
	return m.data, m.err
}

type mockEmail struct {
	err          error
	sentTo       []string
	sentCity     []string
	sentForecast []models.WeatherData
	wg           *sync.WaitGroup
	mu           sync.Mutex
}

func (m *mockEmail) SendWeather(to, city string, forecast models.WeatherData) error {
	if m.wg != nil {
		defer m.wg.Done()
	}
	m.mu.Lock()
	m.sentTo = append(m.sentTo, to)
	m.sentCity = append(m.sentCity, city)
	m.sentForecast = append(m.sentForecast, forecast)
	m.mu.Unlock()
	return m.err
}

func Test_sendOne_Success(t *testing.T) {
	const (
		city  = "Kyiv"
		email = "user@kyiv.ua"
	)
	sub := models.Subscription{ID: 1, City: city, Email: email}

	mockR := &mockRepo{}
	mockW := &mockWeather{
		data: models.WeatherData{City: city, Temperature: 5.0, Condition: "Sunny"},
	}
	mockE := &mockEmail{}

	n := notifier.New(mockR, mockW, mockE, log.New(io.Discard, "", 0), "@every 1h", "0 0 9 * * *")

	err := n.SendOne(context.Background(), sub)
	assert.NoError(t, err)

	assert.Equal(t, []string{city}, mockW.calledWith)
	assert.Equal(t, []string{email}, mockE.sentTo)
	assert.Equal(t, []string{city}, mockE.sentCity)
	assert.Equal(t,
		[]models.WeatherData{{City: city, Temperature: 5.0, Condition: "Sunny"}},
		mockE.sentForecast,
	)
	assert.Equal(t, []int{1}, mockR.updatedIDs)
}

func Test_sendOne_Errors(t *testing.T) {
	const city = "Lviv"

	baseSub := models.Subscription{ID: 2, City: city, Email: "x"}

	mockR1 := &mockRepo{}
	mockW1 := &mockWeather{err: errors.New("api down")}
	mockE1 := &mockEmail{wg: &sync.WaitGroup{}}

	n1 := notifier.New(mockR1, mockW1, mockE1, log.New(io.Discard, "", 0), "@every 1h", "0 0 9 * * *")
	err1 := n1.SendOne(context.Background(), baseSub)
	assert.Error(t, err1, "should return an error on incorrect GetByCity")
	assert.Empty(t, mockE1.sentTo, "must not send email on GetByCity error")
	assert.Empty(t, mockR1.updatedIDs, "must not update LastSent on GetByCity error")

	mockR2 := &mockRepo{}
	mockW2 := &mockWeather{data: models.WeatherData{City: city}}
	wg2 := &sync.WaitGroup{}
	wg2.Add(1)
	mockE2 := &mockEmail{err: errors.New("smtp fail"), wg: wg2}

	n2 := notifier.New(mockR2, mockW2, mockE2, log.New(io.Discard, "", 0), "@every 1h", "0 0 9 * * *")
	err2 := n2.SendOne(context.Background(), baseSub)
	wg2.Wait()
	assert.Error(t, err2)
	assert.Empty(t, mockR2.updatedIDs)
}

func Test_runDue_Success(t *testing.T) {
	const city1 = "Odesa"
	const city2 = "Kharkiv"

	subs := []models.Subscription{
		{ID: 10, City: city1, Email: "a"},
		{ID: 20, City: city2, Email: "b"},
	}

	mockR := &mockRepo{subs: subs}
	mockW := &mockWeather{
		data: models.WeatherData{City: "ignored"},
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(subs))
	mockE := &mockEmail{wg: wg}

	n := notifier.New(mockR, mockW, mockE, log.New(io.Discard, "", 0), "@every 1h", "0 0 9 * * *")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	n.RunDue(ctx, freqTest)

	wg.Wait()

	assert.ElementsMatch(t,
		[]string{city1, city2},
		mockW.calledWith,
		"should call GetByCity for each subscription",
	)
	assert.ElementsMatch(t,
		[]string{"a", "b"},
		mockE.sentTo,
		"should send emails to each subscription",
	)
	assert.ElementsMatch(t,
		[]int{10, 20},
		mockR.updatedIDs,
		"should update LastSent for each subscription",
	)
}

func Test_runDue_FetchError(t *testing.T) {
	mockR := &mockRepo{getErr: errors.New("db down")}
	wg := &sync.WaitGroup{}
	mockE := &mockEmail{wg: wg}
	mockW := &mockWeather{}

	n := notifier.New(mockR, mockW, mockE, log.New(io.Discard, "", 0), "@every 1h", "0 0 9 * * *")

	n.RunDue(context.Background(), freqTest)
	assert.Empty(t, mockR.updatedIDs)
}
