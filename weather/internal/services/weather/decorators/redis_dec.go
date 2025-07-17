package decorators

import (
	"context"
	"fmt"
	"log"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
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
	logger *log.Logger
}

func NewCachedService(
	inner weatherGetterService,
	cache cacheClient[models.WeatherData],
	logger *log.Logger,
) *CachedService {
	return &CachedService{inner: inner, cache: cache, logger: logger}
}

func (s *CachedService) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	key := fmt.Sprintf("weather:%s", city)
	var weather models.WeatherData

	if weather, err := s.cache.Get(ctx, key); err == nil {
		s.logger.Printf("Cache hit for city %s", city)
		return weather, nil
	}

	s.logger.Printf("Cache miss for city %s", city)
	weather, err := s.inner.GetByCity(ctx, city)
	if err != nil {
		return models.WeatherData{}, err
	}

	err = s.cache.Set(ctx, key, weather)
	if err != nil {
		s.logger.Printf("Cache error %s for city %s", err, city)
	}

	return weather, nil
}
