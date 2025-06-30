package decorators

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
)

type weatherGetterService interface {
	GetByCity(ctx context.Context, city string) (models.WeatherData, error)
}

type cacheClient[T any] interface {
	Set(ctx context.Context, key string, value T, expiration time.Duration) error
	Get(ctx context.Context, key string, returnValue *T) error
}

type CachedService struct {
	inner    weatherGetterService
	cache    cacheClient[models.WeatherData]
	logger   *log.Logger
	liveTime time.Duration
}

func NewCachedService(
	inner weatherGetterService,
	cache cacheClient[models.WeatherData],
	logger *log.Logger,
	liveTime time.Duration,
) *CachedService {
	return &CachedService{inner: inner, cache: cache, logger: logger, liveTime: liveTime}
}

func (s *CachedService) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	key := fmt.Sprintf("weather:%s", city)
	var weather models.WeatherData

	if err := s.cache.Get(ctx, key, &weather); err == nil {
		s.logger.Printf("Cache hit for city %s", city)
		return weather, nil
	}

	s.logger.Printf("Cache miss for city %s", city)
	weather, err := s.inner.GetByCity(ctx, city)
	if err != nil {
		return models.WeatherData{}, err
	}

	err = s.cache.Set(ctx, key, weather, s.liveTime)
	if err != nil {
		s.logger.Printf("Cache error for city %s", city)
	}

	return weather, nil
}
