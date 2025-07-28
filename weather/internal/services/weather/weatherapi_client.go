package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
)

type ClientWeatherAPI struct {
	APIKey string
	client HTTPClient
	logger *log.Logger
	apiURL string
}

func NewClientWeatherAPI(apiKey, apiURL string, httpClient HTTPClient, logger *log.Logger) *ClientWeatherAPI {
	return &ClientWeatherAPI{APIKey: apiKey, client: httpClient, logger: logger, apiURL: apiURL}
}

func (s *ClientWeatherAPI) Fetch(ctx context.Context, city string) (models.WeatherData, error) {
	url := fmt.Sprintf(s.apiURL+"?key=%s&q=%s", s.APIKey, city)

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
		return models.WeatherData{}, fmt.Errorf("weather API error: status %s", resp.Status)
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
