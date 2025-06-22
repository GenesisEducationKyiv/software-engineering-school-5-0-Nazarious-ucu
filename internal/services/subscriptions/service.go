package subscriptions

import (
	"crypto/rand"
	"encoding/hex"
)

const bytesNum = 16

type ConfirmationEmailer interface {
	SendConfirmation(email, token string) error
}

type SubscriptionRepository interface {
	Create(email, city, token, frequency string) error
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

func (s *Service) Subscribe(email, city string, frequency string) error {
	tokenBytes := make([]byte, bytesNum)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)

	if err := s.repo.Create(email, city, token, frequency); err != nil {
		return err
	}

	return s.emailer.SendConfirmation(email, token)
}

func (s *Service) Confirm(token string) (bool, error) {
	return s.repo.Confirm(token)
}

func (s *Service) Unsubscribe(token string) (bool, error) {
	return s.repo.Unsubscribe(token)
}
