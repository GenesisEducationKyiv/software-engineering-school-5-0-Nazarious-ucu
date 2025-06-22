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
	Dialect        string `envconfig:"DB_DIALECT"    		 default:"sqlite"`
	Source         string `envconfig:"DB_NAME"    	 		 default:"subscriptions.db"`
	MigrationsPath string `envconfig:"DB_MIGRATIONS_DIR"     default:"./migrations"`
}

type Config struct {
	WeatherAPIKey string `envconfig:"WEATHER_API_KEY" required:"true"`
	Server        Server
	Email         Email
	DB     Db

	TemplatesDir string `envconfig:"TEMPLATES_DIR"    default:"./../../internal/templates"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	return &cfg, nil
}