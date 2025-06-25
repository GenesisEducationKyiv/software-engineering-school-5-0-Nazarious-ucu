package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Server struct {
	Address     string `envconfig:"SERVER_ADDRESS" default:":8080"`
	ReadTimeout int    `envconfig:"SERVER_TIMEOUT" default:"10"`
}

type Email struct {
	User     string `envconfig:"EMAIL_USER"     required:"true"`
	Host     string `envconfig:"EMAIL_HOST"     required:"true"`
	Port     string `envconfig:"EMAIL_PORT"     required:"true"`
	Password string `envconfig:"EMAIL_PASSWORD" required:"true"`
	From     string `envconfig:"EMAIL_FROM"     required:"true"`
}

type Config struct {
	WeatherAPIKey        string `envconfig:"WEATHER_API_KEY" required:"true"`
	OpenWeatherMapAPIKey string `envconfig:"OPEN_WEATHER_MAP_API_KEY" required:"true"`
	WeatherBitAPIKey     string `envconfig:"WEATHER_BIT_API_KEY" required:"true"`
	Server               Server
	Email                Email
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
