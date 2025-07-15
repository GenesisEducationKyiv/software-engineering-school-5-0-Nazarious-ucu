package cfg

import (
	"github.com/kelseyhightower/envconfig"
)

type Server struct {
	Address     string `envconfig:"GATEWAY_SERVER_ADDRESS" default:":8081"`
	ReadTimeout int    `envconfig:"GATEWAY_SERVER_TIMEOUT" default:"10"`
}

type Config struct {
	Server      Server
	SubAddr     string `envconfig:"SUB_ADDR" default:":8081"`
	WeatherAddr string `envconfig:"WEATHER_ADDR" default:":8082"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
