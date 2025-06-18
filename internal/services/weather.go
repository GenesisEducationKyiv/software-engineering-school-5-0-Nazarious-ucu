package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type WeatherData struct {
	City        string  `json:"city"`
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
}

type WeatherService struct {
	APIKey string
	client HTTPClient
}

//const timeoutTime = 10 * time.Second

func NewWeatherService(apiKey string, client HTTPClient) *WeatherService {
	return &WeatherService{APIKey: apiKey, client: client}
}

func (s *WeatherService) GetByCity(ctx context.Context, city string) (WeatherData, error) {
	fmt.Println("Getting weather with API token: ", s.APIKey)
	url := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s", s.APIKey, city)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return WeatherData{}, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return WeatherData{}, err
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Println("failed to close response body: %w", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return WeatherData{}, fmt.Errorf("weather API error: status %d", resp.StatusCode)
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
		return WeatherData{}, err
	}

	return WeatherData{
		City:        raw.Location.Name,
		Temperature: raw.Current.TempC,
		Condition:   raw.Current.Condition.Text,
	}, nil
}
