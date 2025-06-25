package weather

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
)

var errorResponse = errors.New("all weather API clients failed to fetch data")

type client interface {
	Fetch(ctx context.Context, city string) (models.WeatherData, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type ServiceProvider struct {
	logger  *log.Logger
	clients []client
}

func NewService(logger *log.Logger, clients ...client) *ServiceProvider {
	return &ServiceProvider{clients: clients, logger: logger}
}

func (s *ServiceProvider) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	for _, client := range s.clients {
		data, err := client.Fetch(ctx, city)
		if err != nil {
			s.logger.Printf("%v", err)
			continue
		}
		return data, nil
	}
	return models.WeatherData{}, errorResponse
}
