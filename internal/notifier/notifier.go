package notifier

import (
	"context"
	"log"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/repository"
	service "github.com/Nazarious-ucu/weather-subscription-api/internal/services"
)

const (
	freqHourly = "hourly"
	freqDaily  = "daily"
	dayHours   = 24
	sleepTime  = 5 * time.Minute
)

type SubscriptionRepositor interface {
	GetConfirmed() ([]repository.Subscription, error)
	UpdateLastSent(subscriptionID int) error
}

type EmailSender interface {
	SendWeather(to, city string, forecast service.WeatherData) error
}

type Notifier struct {
	Repo           SubscriptionRepositor
	WeatherService handlers.WeatherServicer
	EmailService   EmailSender
}

func NewNotifier(repo SubscriptionRepositor,
	weatherService handlers.WeatherServicer, emailService EmailSender) *Notifier {
	return &Notifier{
		Repo:           repo,
		WeatherService: weatherService,
		EmailService:   emailService,
	}
}

func (n *Notifier) StartWeatherNotifier() {
	go func() {
		for {
			log.Println("Checking for subscriptions to send weather updates")
			subs, err := n.Repo.GetConfirmed()
			if err != nil {
				log.Println("DB query error:", err)
				time.Sleep(time.Minute)
				continue
			}

			now := time.Now()
			for _, sub := range subs {
				if n.shouldSendUpdate(sub, now) {
					err := n.sendWeatherUpdate(sub)
					if err != nil {
						log.Println("DB query error:", err)
					}
				}
			}

			time.Sleep(sleepTime)
		}
	}()
}

func (n *Notifier) shouldSendUpdate(sub repository.Subscription, now time.Time) bool {
	if sub.LastSentAt == nil {
		return true
	}

	var nextTime time.Time
	switch sub.Frequency {
	case freqHourly:
		nextTime = sub.LastSentAt.Add(time.Hour)
	case freqDaily:
		nextTime = sub.LastSentAt.Add(dayHours * time.Hour)
	default:
		return false
	}

	return now.After(nextTime)
}

func (n *Notifier) sendWeatherUpdate(sub repository.Subscription) error {
	ctx := context.Background()

	forecast, err := n.WeatherService.GetByCity(ctx, sub.City)
	if err != nil {
		log.Println("Weather fetch error for", sub.City, ":", err)
		return err
	}

	if err := n.EmailService.SendWeather(sub.Email, sub.City, forecast); err != nil {
		log.Println("Email error:", err)
		return err
	}

	return n.Repo.UpdateLastSent(sub.ID)
}
