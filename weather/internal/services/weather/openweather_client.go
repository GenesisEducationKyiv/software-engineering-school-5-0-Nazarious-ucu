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

type apiResponse struct {
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Pressure  int     `json:"pressure"`
		Humidity  int     `json:"humidity"`
	} `json:"main"`
	Weather []struct {
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
}

// ClientOpenWeatherMap fetches weather data from OpenWeatherMap API.
type ClientOpenWeatherMap struct {
	APIKey string
	apiURL string
	client HTTPClient
	logger zerolog.Logger
}

// NewClientOpenWeatherMap constructs a new OpenWeatherMap client.
func NewClientOpenWeatherMap(apiKey, apiURL string,
	httpClient HTTPClient, logger zerolog.Logger,
) *ClientOpenWeatherMap {
	return &ClientOpenWeatherMap{APIKey: apiKey, apiURL: apiURL, client: httpClient, logger: logger}
}

// Fetch retrieves weather data for a given city, with structured logging.
func (s *ClientOpenWeatherMap) Fetch(ctx context.Context, city string) (models.WeatherData, error) {
	start := time.Now()
	// Build URL
	url := fmt.Sprintf("%s?q=%s&appid=%s&units=metric", s.apiURL, city, s.APIKey)

	s.logger.Debug().
		Str("city", city).
		Str("url", url).
		Msg("starting OpenWeatherMap request")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("city", city).
			Str("url", url).
			Msg("failed to create HTTP request")
		return models.WeatherData{}, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("city", city).
			Str("url", url).
			Msg("error sending HTTP request to OpenWeatherMap")
		return models.WeatherData{}, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			s.logger.Error().
				Err(cerr).
				Str("city", city).
				Msg("failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		s.logger.Error().
			Str("city", city).
			Str("status", resp.Status).
			Msg("OpenWeatherMap API returned non-200 status")
		return models.WeatherData{}, fmt.Errorf("OpenWeatherAPI error: status %s", resp.Status)
	}

	var raw apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		s.logger.Error().
			Err(err).
			Str("city", city).
			Msg("failed to decode OpenWeatherMap response")
		return models.WeatherData{}, err
	}

	data := models.WeatherData{
		City:        city,
		Temperature: raw.Main.Temp,
		Condition:   raw.Weather[0].Main,
	}

	duration := time.Since(start)
	s.logger.Info().
		Str("city", city).
		Dur("duration_ms", duration).
		Msg("successfully fetched weather data")

	return data, nil
}
