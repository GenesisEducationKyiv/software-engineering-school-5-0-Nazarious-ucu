package notifier

import (
	"WeatherSubscriptionAPI/internal/repository"
	"WeatherSubscriptionAPI/internal/services"
	"log"
	"strconv"
	"time"
)

func StartWeatherNotifier(repo *repository.SubscriptionRepository, serviceWeather *service.WeatherService, serviceEmail *service.EmailService) {
	go func() {
		for {
			log.Println("Checking for subscriptions to send weather updates")
			subs, err := repo.GetConfirmedSubscriptions()
			if err != nil {
				log.Println("DB query error:", err)
				time.Sleep(1 * time.Minute)
				continue
			}

			now := time.Now()
			for _, sub := range subs {
				var nextTime time.Time
				if sub.LastSentAt != nil {
					switch sub.Frequency {
					case "hourly":
						nextTime = sub.LastSentAt.Add(time.Hour)
					case "daily":
						nextTime = sub.LastSentAt.Add(24 * time.Hour)
					default:
						continue
					}
				} else {
					nextTime = time.Time{}
				}

				if now.After(nextTime) {
					forecast, err := serviceWeather.GetWeather(sub.City)
					if err != nil {
						log.Println("Weather fetch error for", sub.City, ":", err)
						continue
					}

					temp := strconv.FormatFloat(forecast.Temperature, 'f', 1, 64)

					body := "Weather update for " + sub.City + ":\n" +
						"Temperature: " + temp + "°C\n" +
						"Condition: " + forecast.Condition

					err = serviceEmail.Send(sub.Email, "Your weather update", body)
					if err != nil {
						log.Println("Email error:", err)
						continue
					}

					err = repo.UpdateLastSent(sub.ID)
					if err != nil {
						return
					}
				}
			}

			time.Sleep(5 * time.Minute)
		}
	}()
}
