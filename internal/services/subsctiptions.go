package service

import (
	"WeatherSubscriptionAPI/internal/repository"
	"crypto/rand"
	"encoding/hex"
)

const bytesNum = 16

type SubscriptionService struct {
	Repo    *repository.SubscriptionRepository
	Service *EmailService
}

func NewSubscriptionService(repo *repository.SubscriptionRepository,
	emailService *EmailService) *SubscriptionService {
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

	return s.Service.SendConfirmationEmail(email, token)
}

func (s *SubscriptionService) Confirm(token string) (bool, error) {
	return s.Repo.Confirm(token)
}

func (s *SubscriptionService) Unsubscribe(token string) (bool, error) {
	return s.Repo.Unsubscribe(token)
}
