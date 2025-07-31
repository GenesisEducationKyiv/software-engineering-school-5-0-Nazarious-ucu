package weather

import (
	"context"
	"time"

	weatherpb "github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/weather"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/metrics"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/models"
	"github.com/rs/zerolog"
)

// GrpcWeatherAdapter wraps the gRPC client with structured logging and metrics.
type GrpcWeatherAdapter struct {
	inner  weatherpb.WeatherServiceClient
	logger zerolog.Logger
	m      *metrics.Metrics
}

// NewGrpcWeatherAdapter constructs a new adapter with logger context and metrics collector.
func NewGrpcWeatherAdapter(
	client weatherpb.WeatherServiceClient,
	logger zerolog.Logger,
	m *metrics.Metrics,
) *GrpcWeatherAdapter {
	// enrich logger with component
	logger = logger.With().Str("component", "GrpcWeatherAdapter").Logger()
	return &GrpcWeatherAdapter{inner: client, logger: logger, m: m}
}

// GetByCity retrieves weather data for a given city via gRPC, logging and recording metrics.
func (g *GrpcWeatherAdapter) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	start := time.Now()

	g.logger.Debug().Str("city", city).Msg("calling weather service gRPC method GetByCity")

	resp, err := g.inner.GetByCity(ctx, &weatherpb.WeatherRequest{City: city})
	dur := time.Since(start)

	if err != nil {
		// log error with context
		g.logger.Error().Err(err).
			Str("city", city).
			Dur("duration", dur).
			Msg("weather gRPC call failed")
		return models.WeatherData{}, err
	}

	// on success
	g.logger.Info().
		Str("city", city).
		Dur("duration", dur).
		Msg("weather gRPC call succeeded")

	return models.WeatherData{
		City:        resp.City,
		Temperature: resp.Temperature,
		Condition:   resp.Condition,
	}, nil
}
