package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Email struct {
	User     string `envconfig:"EMAIL_USER"     required:"true"`
	Host     string `envconfig:"EMAIL_HOST"     required:"true"`
	Port     string `envconfig:"EMAIL_PORT"     required:"true"`
	Password string `envconfig:"EMAIL_PASSWORD" required:"true"`
	From     string `envconfig:"EMAIL_FROM"     required:"true"`
}

type RabbitMQ struct {
	Host string `envconfig:"RABBITMQ_HOST" required:"true"`
	Port string `envconfig:"RABBITMQ_PORT" required:"true"`
	User string `envconfig:"RABBITMQ_USER" required:"true"`
	Pass string `envconfig:"RABBITMQ_PASSWORD" required:"true"`
}
type Config struct {
	Email    Email
	RabbitMQ RabbitMQ

	TemplatesDir string `envconfig:"TEMPLATES_DIR"    default:"../../internal/templates"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *RabbitMQ) Address() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/", r.User, r.Pass, r.Host, r.Port)
}
