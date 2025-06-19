package emailer

import (
	"log"
	"net/smtp"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
)

type SMTPService struct {
	User     string
	Host     string
	Port     string
	Password string
	From     string
}

func NewSMTPService(cfg *config.Config) *SMTPService {
	svc := &SMTPService{
		User:     cfg.User,
		Host:     cfg.Host,
		Port:     cfg.Port,
		Password: cfg.Password,
		From:     cfg.From,
	}

	if svc.Host == "" || svc.Port == "" || svc.From == "" {
		log.Printf("SMTP credentials are not fully set: %+v\n", svc)
		return nil
	}
	return svc
}

func (e *SMTPService) Send(to, subject, additionalHeaders, body string) error {
	if e.Host == "" || e.Port == "" {
		log.Println("SMTP credentials are invalid")
	}

	auth := smtp.PlainAuth("", e.User, e.Password, e.Host)

	msg := "From: " + e.From + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n" +
		additionalHeaders + "\n\n" +
		body

	addr := e.Host + ":" + e.Port
	return smtp.SendMail(addr, auth, e.From, []string{to}, []byte(msg))
}
