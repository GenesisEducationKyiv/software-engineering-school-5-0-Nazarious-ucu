package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
)

type bitWeatherAPIResponse = struct {
	Data []struct {
		CityName string  `json:"city_name"`
		Temp     float64 `json:"temp"`
		Weather  struct {
			Description string `json:"description"`
		} `json:"weather"`
	} `json:"data"`
}

type ClientWeatherBit struct {
	APIKey string
	client HTTPClient
	logger *log.Logger
}

func NewWeatherBitClient(apiKey string, httpClient HTTPClient, logger *log.Logger) *ClientWeatherBit {
	return &ClientWeatherBit{APIKey: apiKey, client: httpClient, logger: logger}
}

func (s *ClientWeatherBit) Fetch(ctx context.Context, city string) (models.WeatherData, error) {
	fmt.Println("Getting weather with API token: ", s.APIKey)
	url := fmt.Sprintf("https://api.weatherbit.io/v2.0/current?city=%s&key=%s", city, s.APIKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return models.WeatherData{}, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return models.WeatherData{}, err
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			s.logger.Println("failed to close response body: %w", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return models.WeatherData{}, fmt.Errorf("weather API error: status %d", resp.StatusCode)
	}
	var raw bitWeatherAPIResponse

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return models.WeatherData{}, err
	}

	return models.WeatherData{
		City:        city,
		Temperature: raw.Data[0].Temp,
		Condition:   raw.Data[0].Weather.Description,
	}, nil
}
