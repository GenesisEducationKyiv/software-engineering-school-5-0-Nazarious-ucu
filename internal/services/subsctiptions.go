package service

import (
	"crypto/rand"
	"encoding/hex"
)

const bytesNum = 16

type Emailer interface {
	SendConfirmation(email, token string) error
	Send(to, subject, body string) error
}

type SubscriptionRepository interface {
	Create(email, city, token, frequency string) error
	Confirm(token string) (bool, error)
	Unsubscribe(token string) (bool, error)
}

type SubscriptionService struct {
	Repo    SubscriptionRepository
	Service Emailer
}

func NewSubscriptionService(repo SubscriptionRepository,
	emailService Emailer,
) *SubscriptionService {
	return &SubscriptionService{
		Repo:    repo,
		Service: emailService,
	}
}

func (s *SubscriptionService) Subscribe(email, city string, frequency string) error {
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

func (s *SubscriptionService) Confirm(token string) (bool, error) {
	return s.Repo.Confirm(token)
}

func (s *SubscriptionService) Unsubscribe(token string) (bool, error) {
	return s.Repo.Unsubscribe(token)
}
