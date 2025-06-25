package weather

import (
	"context"
	"log"
	"net/http"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
)

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
	var currentErr error
	for _, client := range s.clients {
		data, err := client.Fetch(ctx, city)
		if err != nil {
			currentErr = err
			s.logger.Printf("%v", err)
			continue
		}
		return data, nil
	}
	return models.WeatherData{}, currentErr
}
