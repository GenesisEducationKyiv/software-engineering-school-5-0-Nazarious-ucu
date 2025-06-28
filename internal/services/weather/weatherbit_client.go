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
	apiURL string
	client HTTPClient
	logger *log.Logger
}

func NewWeatherBitClient(apiKey, apiURL string, httpClient HTTPClient, logger *log.Logger) *ClientWeatherBit {
	return &ClientWeatherBit{APIKey: apiKey, apiURL: apiURL, client: httpClient, logger: logger}
}

func (s *ClientWeatherBit) Fetch(ctx context.Context, city string) (models.WeatherData, error) {
	url := fmt.Sprintf("%s?city=%s&key=%s", s.apiURL, city, s.APIKey)

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

	s.logger.Printf("response code: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return models.WeatherData{}, fmt.Errorf("WeatherBit API error: status %s", resp.Status)
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
