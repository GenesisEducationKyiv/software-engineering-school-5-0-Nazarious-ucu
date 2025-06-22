package notifier

import (
	"context"
	"log"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"
	"github.com/robfig/cron/v3"
)

const (
	timeoutDuration = 30 * time.Second

	freqHourly = "hourly"
	freqDaily  = "daily"
)

type subscriptionRepository interface {
	GetConfirmed() ([]models.Subscription, error)

	GetConfirmedByFrequency(frequency string) ([]models.Subscription, error)
	UpdateLastSent(subscriptionID int) error
}

type emailSender interface {
	SendWeather(to, city string, forecast models.WeatherData) error
}

type weatherGetter interface {
	GetByCity(ctx context.Context, city string) (models.WeatherData, error)
}

type Notifier struct {
	repo           subscriptionRepository
	weatherService weatherGetter
	emailService   emailSender
	logger         *log.Logger
	cron           *cron.Cron
	cancel         context.CancelFunc
}

func New(repo subscriptionRepository, ws weatherGetter,
	es emailSender, logger *log.Logger,
) *Notifier {
	c := cron.New(cron.WithSeconds())
	return &Notifier{
		repo:           repo,
		weatherService: ws,
		emailService:   es,
		logger:         logger,
		cron:           c,
	}
}

func (n *Notifier) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	n.cancel = cancel

	_, err := n.cron.AddFunc("@every 1h", func() {
		n.runDue(ctx, freqHourly)
	})
	if err != nil {
		return
	}

	_, err = n.cron.AddFunc("0 0 9 * * *", func() {
		n.runDue(ctx, freqDaily)
	})
	if err != nil {
		return
	}

	n.cron.Start()
	n.logger.Println("Weather notifier started")
}

func (n *Notifier) Stop() {
	n.cancel()
	stopCtx := n.cron.Stop()
	<-stopCtx.Done()
	n.logger.Println("All cron jobs finished, notifier stopped")
}

func (n *Notifier) runDue(ctx context.Context, frequency string) {
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	subs, err := n.repo.GetConfirmedByFrequency(frequency)
	if err != nil {
		n.logger.Println("Error fetching due subs:", err)
		return
	}

	for _, sub := range subs {
		go func(s models.Subscription) {
			if err := n.sendOne(ctx, s); err != nil {
				n.logger.Println("Error sending update:", err)
			}
		}(sub)
	}
}

func (n *Notifier) sendOne(ctx context.Context, sub models.Subscription) error {
	forecast, err := n.weatherService.GetByCity(ctx, sub.City)
	if err != nil {
		return err
	}

	if err := n.emailService.SendWeather(sub.Email, sub.City, forecast); err != nil {
		return err
	}

	return n.repo.UpdateLastSent(sub.ID)
}
