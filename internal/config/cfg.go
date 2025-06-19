package config

import (
	"os"
	"strconv"
)

type Server struct {
	Address     string
	ReadTimeout int
}

type Db struct {
	Dialect        string
	Source         string
	MigrationsPath string
}
type Config struct {
	WeatherAPIKey string

	User     string
	Host     string
	Port     string
	Password string
	From     string

	Server Server
	DB     Db

	TemplatesDir string
}

func NewConfig() *Config {
	timeout, err := strconv.Atoi(os.Getenv("SERVER_TIMEOUT"))
	if err != nil {
		timeout = 10
	}
	return &Config{
		WeatherAPIKey: os.Getenv("WEATHER_API_KEY"),

		User:     os.Getenv("EMAIL_USER"),
		Host:     os.Getenv("EMAIL_HOST"),
		Port:     os.Getenv("EMAIL_PORT"),
		Password: os.Getenv("EMAIL_PASSWORD"),
		From:     os.Getenv("EMAIL_FROM"),

		Server: Server{
			Address:     os.Getenv("SERVER_ADDRESS"),
			ReadTimeout: timeout,
		},
		DB: Db{
			Dialect:        os.Getenv("DB_DIALECT"),
			Source:         os.Getenv("DB_NAME"),
			MigrationsPath: os.Getenv("DB_MIGRATIONS_DIR"),
		},
		TemplatesDir: os.Getenv("TEMPLATES_DIR"),
	}
}
