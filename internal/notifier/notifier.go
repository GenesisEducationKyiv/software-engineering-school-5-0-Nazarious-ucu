package notifier

import (
	"context"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers"
	"github.com/Nazarious-ucu/weather-subscription-api/internal/repository"
	service "github.com/Nazarious-ucu/weather-subscription-api/internal/services"
	"log"
	"strconv"
	"time"
)

const (
	freqHourly = "hourly"
	freqDaily  = "daily"
	dayHours   = 24
	sleepTime  = 5 * time.Minute
)

type SubscriptionRepositor interface {
	GetConfirmedSubscriptions() ([]repository.Subscription, error)
	UpdateLastSent(subscriptionID int) error
}

type Notifier struct {
	Repo           SubscriptionRepositor
	WeatherService handlers.WeatherServicer
	EmailService   service.Emailer
}

func NewNotifier(repo SubscriptionRepositor, weatherService handlers.WeatherServicer, emailService service.Emailer) *Notifier {
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
			subs, err := n.Repo.GetConfirmedSubscriptions()
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

	forecast, err := n.WeatherService.GetWeather(ctx, sub.City)
	if err != nil {
		log.Println("Weather fetch error for", sub.City, ":", err)
		return err
	}

	temp := strconv.FormatFloat(forecast.Temperature, 'f', 1, 64)
	body := "Weather update for " + sub.City + ":\n" +
		"Temperature: " + temp + "°C\n" +
		"Condition: " + forecast.Condition

	if err := n.EmailService.Send(sub.Email, "Your weather update", body); err != nil {
		log.Println("Email error:", err)
		return err
	}

	return n.Repo.UpdateLastSent(sub.ID)
}
