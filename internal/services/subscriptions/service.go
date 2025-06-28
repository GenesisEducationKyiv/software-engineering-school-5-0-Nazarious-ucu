package subscriptions

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/subscription"
)

const bytesNum = 16

type ConfirmationEmailer interface {
	SendConfirmation(email, token string) error
}

type SubscriptionRepository interface {
	Create(data subscription.UserSubData, token string) error
	Confirm(token string) (bool, error)
	Unsubscribe(token string) (bool, error)
}

type Service struct {
	repo    SubscriptionRepository
	emailer ConfirmationEmailer
}

func NewService(repo SubscriptionRepository,
	emailService ConfirmationEmailer,
) *Service {
	return &Service{
		repo:    repo,
		emailer: emailService,
	}
}

func (s *Service) Subscribe(data subscription.UserSubData) error {
	tokenBytes := make([]byte, bytesNum)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)

	if err := s.repo.Create(data, token); err != nil {
		return err
	}

	return s.emailer.SendConfirmation(data.Email, token)
}

func (s *Service) Confirm(token string) (bool, error) {
	return s.repo.Confirm(token)
}

func (s *Service) Unsubscribe(token string) (bool, error) {
	return s.repo.Unsubscribe(token)
}
