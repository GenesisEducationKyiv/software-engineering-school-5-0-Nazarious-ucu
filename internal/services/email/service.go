package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
)

type Emailer interface {
	Send(to, subject, additionalHeaders, body string) error
}

type Service struct {
	emailer      Emailer
	templatesDir string
}

func NewService(service Emailer, tempsDir string) *Service {
	return &Service{
		emailer:      service,
		templatesDir: tempsDir,
	}
}

func (e *Service) SendConfirmation(toEmail, token string) error {
	tmpl, err := template.ParseFiles(e.templatesDir + "/confirm_email.html")
	if err != nil {
		return err
	}

	var body bytes.Buffer
	err = tmpl.Execute(&body, map[string]string{
		"Email": toEmail,
		"Link":  fmt.Sprintf("http://localhost:8080/confirm/%s", token),
	})
	if err != nil {
		return err
	}

	return e.emailer.Send(toEmail,
		"Confirm Your Weather Subscription",
		"MIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"",
		body.String())
}

func (e *Service) SendWeather(toEmail, city string, forecast models.WeatherData) error {
	temp := strconv.FormatFloat(forecast.Temperature, 'f', 1, 64)
	body := "Weather update for " + city + ":\n" +
		"Temperature: " + temp + "°C\n" +
		"Condition: " + forecast.Condition

	return e.emailer.Send(toEmail, "Your Daily Weather Update", "", body)
}
