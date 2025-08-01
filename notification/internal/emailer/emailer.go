package emailer

import (
	"net/smtp"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/config"
	"github.com/Nazarious-ucu/weather-subscription-api/notification/internal/metrics"
	"github.com/rs/zerolog"
)

// SMTPService wraps smtp.SendMail with structured logging and metrics.
type SMTPService struct {
	user     string
	host     string
	port     string
	password string
	From     string
	logger   zerolog.Logger
	m        metrics.Metrics
}

// NewSMTPService creates an SMTPService with a scoped logger and metrics.
func NewSMTPService(cfg *config.Config, logger zerolog.Logger, m metrics.Metrics) *SMTPService {
	logger = logger.With().Str("component", "SMTPService").Logger()
	return &SMTPService{
		user:     cfg.Email.User,
		host:     cfg.Email.Host,
		port:     cfg.Email.Port,
		password: cfg.Email.Password,
		From:     cfg.Email.From,
		logger:   logger,
		m:        m,
	}
}

// Send sends an email and records logs + metrics.
func (e *SMTPService) Send(to, subject, additionalHeaders, body string) error {
	start := time.Now()
	e.logger.Debug().
		Str("to", to).
		Str("subject", subject).
		Msg("sending email")

	auth := smtp.PlainAuth("", e.user, e.password, e.host)
	msg := "From: " + e.From + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n" +
		additionalHeaders + "\n\n" +
		body
	addr := e.host + ":" + e.port

	err := smtp.SendMail(addr, auth, e.user, []string{to}, []byte(msg))
	duration := time.Since(start)

	if err != nil {
		e.logger.Error().
			Err(err).
			Str("to", to).
			Str("subject", subject).
			Dur("duration", duration).
			Msg("email send failed")
		return err
	}

	e.logger.Info().
		Str("to", to).
		Str("subject", subject).
		Dur("duration", duration).
		Msg("email sent successfully")
	return nil
}
