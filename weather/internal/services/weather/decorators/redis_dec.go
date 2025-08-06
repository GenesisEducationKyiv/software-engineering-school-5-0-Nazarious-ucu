package decorators

import (
	"context"
	"fmt"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
	"github.com/rs/zerolog"
)

type weatherGetterService interface {
	GetByCity(ctx context.Context, city string) (models.WeatherData, error)
}

type cacheClient[T any] interface {
	Set(ctx context.Context, key string, value T) error
	Get(ctx context.Context, key string) (T, error)
}

type CachedService struct {
	inner  weatherGetterService
	cache  cacheClient[models.WeatherData]
	logger zerolog.Logger
}

func NewCachedService(
	inner weatherGetterService,
	cache cacheClient[models.WeatherData],
	logger zerolog.Logger,
) *CachedService {
	return &CachedService{inner: inner, cache: cache, logger: logger}
}

func (s *CachedService) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	key := fmt.Sprintf("weather:%s", city)
	var weather models.WeatherData

	// Try cache
	weather, err := s.cache.Get(ctx, key)
	if err == nil {
		s.logger.Info().
			Ctx(ctx).
			Str("city", city).
			Str("key", key).
			Msg("cache hit")
		return weather, nil
	}
	s.logger.Info().
		Ctx(ctx).
		Str("city", city).
		Str("key", key).
		Err(err).
		Msg("cache miss")

	// Fallback to inner service
	weather, err = s.inner.GetByCity(ctx, city)
	if err != nil {
		s.logger.Error().
			Ctx(ctx).
			Str("city", city).
			Err(err).
			Msg("inner service failed")
		return models.WeatherData{}, err
	}

	// Populate cache
	if err := s.cache.Set(ctx, key, weather); err != nil {
		s.logger.Error().
			Ctx(ctx).
			Str("city", city).
			Str("key", key).
			Err(err).
			Msg("cache set failed")
	}

	return weather, nil
}
