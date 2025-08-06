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

// ClientWeatherAPI fetches weather data from WeatherAPI.com.
type ClientWeatherAPI struct {
	APIKey string
	apiURL string
	client HTTPClient
	logger zerolog.Logger
}

// NewClientWeatherAPI constructs a new WeatherAPI client.
func NewClientWeatherAPI(
	apiKey, apiURL string,
	httpClient HTTPClient,
	logger zerolog.Logger,
) *ClientWeatherAPI {
	return &ClientWeatherAPI{APIKey: apiKey, apiURL: apiURL, client: httpClient, logger: logger}
}

// Fetch retrieves weather data for a given city, with structured logging.
func (s *ClientWeatherAPI) Fetch(ctx context.Context, city string) (models.WeatherData, error) {
	start := time.Now()

	// Build URL
	url := fmt.Sprintf("%s?key=%s&q=%s", s.apiURL, s.APIKey, city)

	s.logger.Debug().
		Ctx(ctx).
		Str("city", city).
		Str("url", url).
		Msg("starting WeatherAPI request")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		s.logger.Error().
			Err(err).
			Ctx(ctx).
			Str("city", city).
			Str("url", url).
			Msg("failed to create HTTP request")
		return models.WeatherData{}, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error().
			Err(err).
			Ctx(ctx).
			Str("city", city).
			Str("url", url).
			Msg("error sending HTTP request to WeatherAPI")
		return models.WeatherData{}, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			s.logger.Error().
				Err(cerr).
				Ctx(ctx).
				Str("city", city).
				Msg("failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		s.logger.Error().
			Ctx(ctx).
			Str("city", city).
			Str("status", resp.Status).
			Msg("WeatherAPI returned non-200 status")
		return models.WeatherData{}, fmt.Errorf("weather API error: status %s", resp.Status)
	}

	// Decode response
	var raw struct {
		Location struct {
			Name string `json:"name"`
		} `json:"location"`
		Current struct {
			TempC     float64 `json:"temp_c"`
			Condition struct {
				Text string `json:"text"`
			} `json:"condition"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		s.logger.Error().
			Ctx(ctx).
			Err(err).
			Str("city", city).
			Msg("failed to decode WeatherAPI response")
		return models.WeatherData{}, err
	}

	data := models.WeatherData{
		City:        raw.Location.Name,
		Temperature: raw.Current.TempC,
		Condition:   raw.Current.Condition.Text,
	}

	duration := time.Since(start)
	s.logger.Info().
		Ctx(ctx).
		Str("city", city).
		Dur("duration_ms", duration).
		Msg("successfully fetched weather data from WeatherAPI")

	return data, nil
}
