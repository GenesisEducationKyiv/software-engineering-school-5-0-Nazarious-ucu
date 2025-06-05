package notifier

import (
	"WeatherSubscriptionAPI/internal/repository"
	service "WeatherSubscriptionAPI/internal/services"
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

func StartWeatherNotifier(repo *repository.SubscriptionRepository, serviceWeather *service.WeatherService, serviceEmail *service.EmailService) {
	go func() {
		for {
			log.Println("Checking for subscriptions to send weather updates")
			subs, err := repo.GetConfirmedSubscriptions()
			if err != nil {
				log.Println("DB query error:", err)
				time.Sleep(time.Minute)
				continue
			}

			now := time.Now()
			for _, sub := range subs {

				if shouldSendUpdate(sub, now) {
					err := sendWeatherUpdate(sub, serviceWeather, serviceEmail, repo)

					if err != nil {
						log.Println("DB query error:", err)
					}
				}
			}

			time.Sleep(sleepTime)
		}
	}()
}

func shouldSendUpdate(sub repository.Subscription, now time.Time) bool {
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

func sendWeatherUpdate(sub repository.Subscription, weatherSvc *service.WeatherService, emailSvc *service.EmailService, repo *repository.SubscriptionRepository) error {
	forecast, err := weatherSvc.GetWeather(sub.City)
	if err != nil {
		log.Println("Weather fetch error for", sub.City, ":", err)
		return err
	}

	temp := strconv.FormatFloat(forecast.Temperature, 'f', 1, 64)
	body := "Weather update for " + sub.City + ":\n" +
		"Temperature: " + temp + "°C\n" +
		"Condition: " + forecast.Condition

	if err := emailSvc.Send(sub.Email, "Your weather update", body); err != nil {
		log.Println("Email error:", err)
		return err
	}

	return repo.UpdateLastSent(sub.ID)
}
