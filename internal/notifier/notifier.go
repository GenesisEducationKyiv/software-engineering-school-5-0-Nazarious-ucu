package notifier

import (
	"context"
	"log"
	"sync"
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
	GetConfirmedByFrequency(frequency string, ctx context.Context) ([]models.Subscription, error)
	UpdateLastSent(subscriptionID int, ctx context.Context) error
}

type emailSender interface {
	SendWeather(to string, forecast models.WeatherData) error
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
	hourlySpec     string
	dailySpec      string
}

func New(repo subscriptionRepository, ws weatherGetter,
	es emailSender, logger *log.Logger, hourlySpec, dailySpec string,
) *Notifier {
	c := cron.New(cron.WithSeconds())
	return &Notifier{
		repo:           repo,
		weatherService: ws,
		emailService:   es,
		logger:         logger,
		cron:           c,
		hourlySpec:     hourlySpec,
		dailySpec:      dailySpec,
	}
}

func (n *Notifier) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	n.cancel = cancel

	_, err := n.cron.AddFunc(n.hourlySpec, func() {
		n.RunDue(ctx, freqHourly)
	})
	if err != nil {
		return
	}

	_, err = n.cron.AddFunc(n.dailySpec, func() {
		n.RunDue(ctx, freqDaily)
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

func (n *Notifier) RunDue(ctx context.Context, frequency string) {
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	subs, err := n.repo.GetConfirmedByFrequency(frequency, ctx)
	if err != nil {
		n.logger.Println("Error fetching due subs:", err)
		return
	}
	wg := &sync.WaitGroup{}

	wg.Add(len(subs))

	for _, sub := range subs {
		go func(s models.Subscription) {
			defer wg.Done()
			if err := n.SendOne(ctx, s); err != nil {
				n.logger.Println("Error sending update:", err)
			}
		}(sub)
	}

	wg.Wait()
}

func (n *Notifier) SendOne(ctx context.Context, sub models.Subscription) error {
	forecast, err := n.weatherService.GetByCity(ctx, sub.City)
	if err != nil {
		return err
	}

	if err := n.emailService.SendWeather(sub.Email, forecast); err != nil {
		return err
	}

	return n.repo.UpdateLastSent(sub.ID, ctx)
}
