package cfg

import (
	"github.com/kelseyhightower/envconfig"
)

type Server struct {
	Address     string `envconfig:"GATEWAY_SERVER_ADDRESS" default:"localhost"`
	Port        string `envconfig:"GATEWAY_SERVER_PORT" default:"8081"`
	ReadTimeout int    `envconfig:"GATEWAY_SERVER_TIMEOUT" default:"10"`
}

type WeatherServer struct {
	Host string `envconfig:"WEATHER_HOST" default:"localhost"`
	Port string `envconfig:"WEATHER_PORT" default:"8080"`
}

type SubServer struct {
	Host string `envconfig:"SUB_HOST" default:"localhost"`
	Port string `envconfig:"SUB_PORT" default:"8082"`
}

type Config struct {
	Server        Server
	SubServer     SubServer
	WeatherServer WeatherServer
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) ServerAddress() string {
	return c.Server.Address + ":" + c.Server.Port
}

func (w WeatherServer) Address() string {
	return w.Host + ":" + w.Port
}

func (s SubServer) Address() string {
	return s.Host + ":" + s.Port
}
