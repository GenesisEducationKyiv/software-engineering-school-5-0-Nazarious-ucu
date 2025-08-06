package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
)

type bitWeatherAPIResponse struct {
	Data []struct {
		CityName string  `json:"city_name"`
		Temp     float64 `json:"temp"`
		Weather  struct {
			Description string `json:"description"`
		} `json:"weather"`
	} `json:"data"`
}

// ClientWeatherBit fetches weather data from WeatherBit API.
type ClientWeatherBit struct {
	APIKey string
	apiURL string
	client HTTPClient
	logger zerolog.Logger
}

// NewClientWeatherBit constructs a new WeatherBit client.
func NewClientWeatherBit(
	apiKey, apiURL string,
	httpClient HTTPClient,
	logger zerolog.Logger,
) *ClientWeatherBit {
	return &ClientWeatherBit{APIKey: apiKey, apiURL: apiURL, client: httpClient, logger: logger}
}

// Fetch retrieves weather data for a given city, with structured logging and timing.
func (s *ClientWeatherBit) Fetch(ctx context.Context, city string) (models.WeatherData, error) {
	start := time.Now()
	// Build request URL
	url := fmt.Sprintf("%s?city=%s&key=%s", s.apiURL, city, s.APIKey)

	s.logger.Debug().
		Ctx(ctx).
		Str("city", city).
		Str("url", url).
		Msg("starting WeatherBit request")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		s.logger.Error().
			Ctx(ctx).
			Err(err).
			Str("city", city).
			Msg("failed to create HTTP request")
		return models.WeatherData{}, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error().
			Ctx(ctx).
			Err(err).
			Str("city", city).
			Msg("error sending HTTP request to WeatherBit")
		return models.WeatherData{}, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			s.logger.Error().
				Ctx(ctx).
				Err(cerr).
				Str("city", city).
				Msg("failed to close response body")
		}
	}()

	s.logger.Debug().
		Ctx(ctx).
		Int("status_code", resp.StatusCode).
		Msg("received response from WeatherBit API")

	if resp.StatusCode != http.StatusOK {
		s.logger.Error().
			Ctx(ctx).
			Str("city", city).
			Int("status_code", resp.StatusCode).
			Msg("WeatherBit API returned non-200 status")
		return models.WeatherData{}, fmt.Errorf("WeatherBit API error: status %s", resp.Status)
	}

	var raw bitWeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		s.logger.Error().
			Err(err).
			Ctx(ctx).
			Str("city", city).
			Msg("failed to decode WeatherBit response")
		return models.WeatherData{}, err
	}

	// Extract first entry
	if len(raw.Data) == 0 {
		s.logger.Error().
			Ctx(ctx).
			Str("city", city).
			Msg("no data in WeatherBit response")
		return models.WeatherData{}, fmt.Errorf("WeatherBit API returned empty data for city %s", city)
	}

	entry := raw.Data[0]
	data := models.WeatherData{
		City:        city,
		Temperature: entry.Temp,
		Condition:   entry.Weather.Description,
	}

	duration := time.Since(start)
	s.logger.Info().
		Ctx(ctx).
		Str("city", city).
		Dur("duration_ms", duration).
		Msg("successfully fetched weather data from WeatherBit")

	return data, nil
}
