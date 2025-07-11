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
		user:     cfg.Email.User,
		host:     cfg.Email.Host,
		port:     cfg.Email.Port,
		password: cfg.Email.Password,
		From:     cfg.Email.From,
		logger:   logger,
	}

	return svc
}

func (e *SMTPService) Send(to, subject, additionalHeaders, body string) error {
	auth := smtp.PlainAuth("", e.user, e.password, e.host)

	msg := "From: " + e.From + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n" +
		additionalHeaders + "\n\n" +
		body

	addr := e.host + ":" + e.port
	return smtp.SendMail(addr, auth, e.user, []string{to}, []byte(msg))
}
