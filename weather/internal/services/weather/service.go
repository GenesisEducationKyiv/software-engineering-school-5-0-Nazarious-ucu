package weather

import (
	"context"
	"errors"
	"github.com/Nazarious-ucu/weather-subscription-api/weather/internal/models"
	"net/http"
	"path"
	"reflect"
	"runtime"

	"github.com/rs/zerolog"
)

type client interface {
	Fetch(ctx context.Context, city string) (models.WeatherData, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type ServiceProvider struct {
	logger  zerolog.Logger
	clients []client
}

func NewService(logger zerolog.Logger, clients ...client) *ServiceProvider {
	return &ServiceProvider{clients: clients, logger: logger}
}

func getFuncName(fn interface{}) string {
	pc := reflect.ValueOf(fn).Pointer()
	return path.Base(runtime.FuncForPC(pc).Name())
}

func (s *ServiceProvider) GetByCity(ctx context.Context, city string) (models.WeatherData, error) {
	for _, cl := range s.clients {
		s.logger.Info().
			Ctx(ctx).
			Str("client", getFuncName(cl.Fetch)).
			Str("city", city).
			Msg("calling Fetch")
		data, err := cl.Fetch(ctx, city)
		if err != nil {
			s.logger.Error().
				Ctx(ctx).
				Str("client", getFuncName(cl.Fetch)).
				Err(err).
				Msg("fetch failed")
			continue
		}
		s.logger.Info().
			Ctx(ctx).
			Str("client", getFuncName(cl.Fetch)).
			Msg("fetch succeeded")
		return data, nil
	}
	err := errors.New("all weather API clients failed")
	s.logger.Fatal().
		Err(err).
		Ctx(ctx).
		Msg("GetByCity giving up")
	return models.WeatherData{}, err
}
