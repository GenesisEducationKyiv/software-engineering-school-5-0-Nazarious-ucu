package config

import "os"

type Config struct {
	WeatherAPIKey string

	User     string
	Host     string
	Port     string
	Password string
	From     string

	Server struct {
		Address string
	}
}

func NewConfig() *Config {
	return &Config{
		WeatherAPIKey: os.Getenv("WEATHER_API_KEY"),

		User:     os.Getenv("EMAIL_USER"),
		Host:     os.Getenv("EMAIL_HOST"),
		Port:     os.Getenv("EMAIL_PORT"),
		Password: os.Getenv("EMAIL_PASSWORD"),
		From:     os.Getenv("EMAIL_FROM"),

		Server: struct {
			Address string
		}{
			Address: os.Getenv("SERVER_ADDRESS"),
		},
	}
}
