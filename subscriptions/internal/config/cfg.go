package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Server struct {
	Host        string `envconfig:"SUB_SERVER_HOST" default:"localhost"`
	GrpcPort    string `envconfig:"SUB_SERVER_GRPC_PORT" default:"8080"`
	HTTPPort    string `envconfig:"SUB_SERVER_HTTP_PORT" default:"50051"`
	ReadTimeout int    `envconfig:"SUB_SERVER_TIMEOUT" default:"10"`
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

type Config struct {
	WeatherRPCAddr string `envconfig:"WEATHER_SERVER_ADDR" default:"localhost"`
	WeatherRPCPort string `envconfig:"WEATHER_SERVER_PORT" default:":8082"`

	Server       Server
	Email        Email
	DB           Db
	NotifierFreq NotifierFrequency

	TemplatesDir string `envconfig:"TEMPLATES_DIR"    default:"../../internal/templates"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) ServerAddress() string {
	return c.Server.Host + ":" + c.Server.HTTPPort
}
