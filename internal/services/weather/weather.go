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

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Service struct {
	APIKey string
	client HTTPClient
	logger *log.Logger
}

func NewService(apiKey string, httpClient HTTPClient, logger *log.Logger) *Service {
	return &Service{APIKey: apiKey, client: httpClient, logger: logger}
}

func (s *Service) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	fmt.Println("Getting weather with API token: ", s.APIKey)
	url := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s", s.APIKey, city)

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
		return models.WeatherData{}, err
	}

	return models.WeatherData{
		City:        raw.Location.Name,
		Temperature: raw.Current.TempC,
		Condition:   raw.Current.Condition.Text,
	}, nil
}
