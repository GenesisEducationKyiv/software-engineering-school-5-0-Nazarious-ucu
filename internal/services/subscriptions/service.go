package subscriptions

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/subscription"
)

const bytesNum = 16

type ConfirmationEmailer interface {
	SendConfirmation(email, token string) error
}

type subscriptionRepository interface {
	Create(ctx context.Context, data subscription.UserSubData, token string) error
	Confirm(ctx context.Context, token string) (bool, error)
	Unsubscribe(ctx context.Context, token string) (bool, error)
}

type Service struct {
	repo    subscriptionRepository
	emailer ConfirmationEmailer
}

func NewService(repo subscriptionRepository,
	emailService ConfirmationEmailer,
) *Service {
	return &Service{
		repo:    repo,
		emailer: emailService,
	}
}

func (s *Service) Subscribe(ctx context.Context, data subscription.UserSubData) error {
	tokenBytes := make([]byte, bytesNum)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)

	if err := s.repo.Create(ctx, data, token); err != nil {
		return err
	}

	return s.emailer.SendConfirmation(data.Email, token)
}

func (s *Service) Confirm(ctx context.Context, token string) (bool, error) {
	return s.repo.Confirm(ctx, token)
}

func (s *Service) Unsubscribe(ctx context.Context, token string) (bool, error) {
	return s.repo.Unsubscribe(ctx, token)
}
