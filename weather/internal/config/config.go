package config

import "github.com/kelseyhightower/envconfig"

type Server struct {
	Port        string `envconfig:"WEATHER_SERVER_PORT" default:":8082"`
	ReadTimeout int    `envconfig:"WEATHER_SERVER_TIMEOUT" default:"10"`
}

type Breaker struct {
	TimeInterval int    `envconfig:"BREAKER_INTERVAL" default:"30"`
	TimeTimeOut  int    `envconfig:"BREAKER_TIMEOUT" default:"10"`
	RepeatNumber uint32 `envconfig:"BREAKER_REPEAT_NUM" default:"5"`
}

type Redis struct {
	Host     string `envconfig:"REDIS_HOST" default:"localhost"`
	Port     string `envconfig:"REDIS_PORT" default:"6379"`
	DbType   int    `envconfig:"REDIS_DB_TYPE" required:"true"`
	LiveTime int    `envconfig:"REDIS_LIVE_TIME" default:"1"`
}

type Config struct {
	WeatherAPIKey string `envconfig:"WEATHER_API_KEY" required:"true"`
	WeatherAPIURL string `envconfig:"WEATHER_API_URL" required:"true"`

	OpenWeatherMapAPIKey string `envconfig:"OPEN_WEATHER_MAP_API_KEY" required:"true"`
	OpenWeatherMapURL    string `envconfig:"OPEN_WEATHER_MAP_URL" required:"true"`

	WeatherBitAPIKey string `envconfig:"WEATHER_BIT_API_KEY" required:"true"`
	WeatherBitURL    string `envconfig:"WEATHER_BIT_URL" required:"true"`

	Server  Server
	Breaker Breaker
	Redis   Redis

	LogsPath string `envconfig:"LOGS_PATH" default:"./log/weather-subscription-api.log"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
