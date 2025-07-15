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

type apiResponse = struct {
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

type ClientOpenWeatherMap struct {
	APIKey string
	apiURL string
	client HTTPClient
	logger *log.Logger
}

func NewClientOpenWeatherMap(apiKey, apiURL string,
	httpClient HTTPClient, logger *log.Logger,
) *ClientOpenWeatherMap {
	return &ClientOpenWeatherMap{APIKey: apiKey, apiURL: apiURL, client: httpClient, logger: logger}
}

func (s *ClientOpenWeatherMap) Fetch(ctx context.Context, city string) (models.WeatherData, error) {
	url := fmt.Sprintf("%s?q=%s&appid=%s&units=metric", s.apiURL, city, s.APIKey)

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
		return models.WeatherData{}, fmt.Errorf("OpenWeatherAPI error: status %s", resp.Status)
	}
	var raw apiResponse

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return models.WeatherData{}, err
	}

	return models.WeatherData{
		City:        city,
		Temperature: raw.Main.Temp,
		Condition:   raw.Weather[0].Main,
	}, nil
}
