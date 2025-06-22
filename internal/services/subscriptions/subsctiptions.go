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
	Repo    SubscriptionRepository
	Service ConfirmationEmailer
}

func NewService(repo SubscriptionRepository,
	emailService ConfirmationEmailer,
) *Service {
	return &Service{
		Repo:    repo,
		Service: emailService,
	}
}

func (s *Service) Subscribe(email, city string, frequency string) error {
	tokenBytes := make([]byte, bytesNum)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)

	if err := s.Repo.Create(email, city, token, frequency); err != nil {
		return err
	}

	return s.Service.SendConfirmation(email, token)
}

func (s *Service) Confirm(token string) (bool, error) {
	return s.Repo.Confirm(token)
}

func (s *Service) Unsubscribe(token string) (bool, error) {
	return s.Repo.Unsubscribe(token)
}
