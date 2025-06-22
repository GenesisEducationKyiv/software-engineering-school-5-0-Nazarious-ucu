package emailer

import (
	"log"
	"net/smtp"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/config"
)

type SMTPService struct {
	user     string
	host     string
	port     string
	password string
	From     string
	logger   *log.Logger
}

func NewSMTPService(cfg *config.Config, logger *log.Logger) *SMTPService {
	svc := &SMTPService{
		user:     cfg.User,
		host:     cfg.Host,
		port:     cfg.Port,
		password: cfg.Password,
		From:     cfg.From,
		logger:   logger,
	}

	if svc.user == "" || svc.host == "" || svc.port == "" || svc.password == "" || svc.From == "" {
		logger.Printf("SMTP credentials are not fully set: %+v\n", svc)
		return nil
	}
	return svc
}

func (e *SMTPService) Send(to, subject, additionalHeaders, body string) error {
	if e.host == "" || e.port == "" || e.user == "" || e.password == "" {
		e.logger.Println("SMTP credentials are invalid")
	}

	auth := smtp.PlainAuth("", e.user, e.password, e.host)

	msg := "From: " + e.From + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n" +
		additionalHeaders + "\n\n" +
		body

	addr := e.host + ":" + e.port
	return smtp.SendMail(addr, auth, e.user, []string{to}, []byte(msg))
}
