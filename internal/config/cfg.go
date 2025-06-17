package config

import (
	"os"
	"strconv"
)

type Server struct {
	Address     string
	ReadTimeout int
}

type Config struct {
	WeatherAPIKey string

	User     string
	Host     string
	Port     string
	Password string
	From     string

	Server Server
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
			ReadTimeout: timeout, // Default read timeout in seconds
		},
	}
}
