package weather

import (
	"context"
	"log"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
	weatherpb "github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/weather"
)

type GrpcWeatherAdapter struct {
	inner  weatherpb.WeatherServiceClient
	logger *log.Logger
}

func NewGrpcWeatherAdapter(client weatherpb.WeatherServiceClient, logger *log.Logger) *GrpcWeatherAdapter {
	return &GrpcWeatherAdapter{
		inner:  client,
		logger: logger,
	}
}

func (g GrpcWeatherAdapter) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	resp, err := g.inner.GetByCity(ctx, &weatherpb.WeatherRequest{City: city})
	if err != nil {
		g.logger.Printf("failed to get weather data for city %s: %v", city, err)
		return models.WeatherData{}, err
	}

	return models.WeatherData{
		City:        resp.City,
		Temperature: resp.Temperature,
		Condition:   resp.Condition,
	}, nil
}
