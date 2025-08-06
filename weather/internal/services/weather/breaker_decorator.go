package weather

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/sony/gobreaker"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
)

// BreakerConfig configures the behavior of the circuit breaker.
type BreakerConfig struct {
	TimeInterval time.Duration
	TimeTimeOut  time.Duration
	RepeatNumber uint32
}

// BreakerClient wraps another weather client with a circuit breaker and structured logging.
type BreakerClient struct {
	cb      *gobreaker.CircuitBreaker
	wrapped client
	logger  zerolog.Logger
}

func NewBreakerClient(name string, cfg BreakerConfig, logger zerolog.Logger, wrapped client) *BreakerClient {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,
		Interval:    cfg.TimeInterval,
		Timeout:     cfg.TimeTimeOut,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cfg.RepeatNumber
		},
	}
	cb := gobreaker.NewCircuitBreaker(settings)
	return &BreakerClient{cb: cb, wrapped: wrapped, logger: logger}
}

// Fetch executes the wrapped client's Fetch under the circuit breaker, logging entry, exit, and errors.
func (b *BreakerClient) Fetch(ctx context.Context, city string) (models.WeatherData, error) {
	start := time.Now()
	b.logger.Debug().
		Ctx(ctx).
		Str("breaker_name", b.cb.Name()).
		Str("city", city).
		Msg("circuit breaker: starting request")

	result, err := b.cb.Execute(func() (interface{}, error) {
		return b.wrapped.Fetch(ctx, city)
	})
	duration := time.Since(start)
	if err != nil {
		b.logger.Error().
			Ctx(ctx).
			Str("breaker_name", b.cb.Name()).
			Str("city", city).
			Dur("duration_ms", duration).
			Err(err).
			Msg("circuit breaker: request failed")
		return models.WeatherData{}, err
	}

	res, ok := result.(models.WeatherData)
	if !ok {
		b.logger.Error().
			Ctx(ctx).
			Str("breaker_name", b.cb.Name()).
			Str("city", city).
			Dur("duration_ms", duration).
			Msg("circuit breaker: unexpected result type")
		return models.WeatherData{}, fmt.Errorf("returned unexpected result type: %T", result)
	}

	b.logger.Info().
		Ctx(ctx).
		Str("breaker_name", b.cb.Name()).
		Str("city", city).
		Dur("duration_ms", duration).
		Msg("circuit breaker: request succeeded")
	return res, nil
}
