package weather

import (
	"context"
	"fmt"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
	"github.com/sony/gobreaker"
)

type BreakerConfig struct {
	TimeInterval time.Duration
	TimeTimeOut  time.Duration
	RepeatNumber uint32
}

type BreakerClient struct {
	name    string
	cb      *gobreaker.CircuitBreaker
	wrapped client
}

func NewBreakerClient(name string, cfg BreakerConfig, wrapped client) *BreakerClient {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,
		Interval:    cfg.TimeInterval,
		Timeout:     cfg.TimeTimeOut,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cfg.RepeatNumber
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
			fmt.Errorf("%s unavailable: %w", b.name, err)
	}
	res, ok := result.(models.WeatherData)
	if !ok {
		return models.WeatherData{},
			fmt.Errorf("%s returned unexpected result", b.name)
	}
	return res, nil
}
