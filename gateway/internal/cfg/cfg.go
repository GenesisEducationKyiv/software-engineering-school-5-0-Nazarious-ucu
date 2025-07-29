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
	Host     string `envconfig:"WEATHER_HOST" default:"localhost"`
	GrpcPort string `envconfig:"WEATHER_GRPC_PORT" default:"50052"`
	HTTPPort string `envconfig:"WEATHER_HTTP_PORT" default:"8082"`
}

type SubServer struct {
	Host     string `envconfig:"SUB_HOST" default:"localhost"`
	GrpcPort string `envconfig:"SUB_PORT" default:"50051"`
	HTTPPort string `envconfig:"SUB_HTTP_PORT" default:"8080"`
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
	return w.Host + ":" + w.GrpcPort
}

func (s SubServer) Address() string {
	return s.Host + ":" + s.GrpcPort
}
