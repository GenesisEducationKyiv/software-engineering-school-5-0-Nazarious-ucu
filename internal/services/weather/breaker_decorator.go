package weather

import (
	"context"
	"errors"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
	"github.com/sony/gobreaker"
)

const (
	timeInterval = time.Duration(30) * time.Second
	timeTimeOut  = time.Duration(15) * time.Second

	repeatNumber = 5
)

type BreakerClient struct {
	name    string
	cb      *gobreaker.CircuitBreaker
	wrapped client
}

func NewBreakerClient(name string, wrapped client) *BreakerClient {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,
		Interval:    timeInterval,
		Timeout:     timeTimeOut,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= repeatNumber
		},
	}
	return &BreakerClient{
		name:    name,
		cb:      gobreaker.NewCircuitBreaker(settings),
		wrapped: wrapped,
	}
}

func (b *BreakerClient) Fetch(ctx context.Context, city string) (models.WeatherData, error) {
	result, err := b.cb.Execute(func() (interface{}, error) {
		return b.wrapped.Fetch(ctx, city)
	})
	if err != nil {
		return models.WeatherData{},
			errors.New(b.name + " unavailable: " + err.Error())
	}
	res, ok := result.(models.WeatherData)
	if !ok {
		return models.WeatherData{},
			errors.New(b.name + " unavailable: ")
	}
	return res, nil
}
