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

type Db struct {
	Dialect        string `envconfig:"DB_DIALECT" default:"sqlite"`
	Source         string `envconfig:"DB_NAME" default:"subscriptions.db"`
	MigrationsPath string `envconfig:"DB_MIGRATIONS_DIR"     default:"./migrations"`
}

type NotifierFrequency struct {
	DailyFrequency  string `envconfig:"NOTIFIER_FREQUENCY" default:"0 0 9 * *"`
	HourlyFrequency string `envconfig:"NOTIFIER_HOURLY_FREQUENCY" default:"0 * * * *"`
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

	Server       Server
	Email        Email
	DB           Db
	NotifierFreq NotifierFrequency
	Breaker      Breaker
	Redis        Redis

	TemplatesDir string `envconfig:"TEMPLATES_DIR"    default:"../../internal/templates"`
	LogsPath     string `envconfig:"LOGS_PATH" default:"./log/weather-subscription-api.log"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
