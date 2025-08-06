package notifier

import (
	"context"
	"sync"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/metrics"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/models"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

const (
	timeoutDuration = 30 * time.Second

	freqHourly = "hourly"
	freqDaily  = "daily"
)

type subscriptionRepository interface {
	GetConfirmedByFrequency(ctx context.Context, frequency string) ([]models.Subscription, error)
	UpdateLastSent(ctx context.Context, subscriptionID int) error
}

type emailSender interface {
	SendWeather(ctx context.Context, to string, forecast models.WeatherData) error
}

type weatherGetter interface {
	GetByCity(ctx context.Context, city string) (models.WeatherData, error)
}

// Notifier schedules and sends weather updates to subscribers.
type Notifier struct {
	repo           subscriptionRepository
	weatherService weatherGetter
	emailService   emailSender
	logger         zerolog.Logger
	cron           *cron.Cron
	cancel         context.CancelFunc
	m              *metrics.Metrics
	hourlySpec     string
	dailySpec      string
}

// New constructs a Notifier with structured logging and metrics.
func New(
	repo subscriptionRepository,
	ws weatherGetter,
	es emailSender,
	logger zerolog.Logger,
	hourlySpec, dailySpec string,
	m *metrics.Metrics,
) *Notifier {
	// enrich logger with component
	logger = logger.With().Str("component", "Notifier").Logger()
	c := cron.New(cron.WithSeconds())
	return &Notifier{
		repo:           repo,
		weatherService: ws,
		emailService:   es,
		logger:         logger,
		cron:           c,
		hourlySpec:     hourlySpec,
		dailySpec:      dailySpec,
		m:              m,
	}
}

// Start schedules the hourly and daily jobs.
func (n *Notifier) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	n.cancel = cancel

	// Schedule hourly job
	if _, err := n.cron.AddFunc(n.hourlySpec, func() { n.RunDue(ctx, freqHourly) }); err != nil {
		n.logger.Error().Err(err).Msg("failed to schedule hourly job")
		n.m.TechnicalErrors.WithLabelValues("cron_schedule_error", "critical").Inc()
		return
	}

	// Schedule daily job
	if _, err := n.cron.AddFunc(n.dailySpec, func() { n.RunDue(ctx, freqDaily) }); err != nil {
		n.logger.Error().Err(err).Msg("failed to schedule daily job")
		n.m.TechnicalErrors.WithLabelValues("cron_schedule_error", "critical").Inc()
		return
	}

	n.cron.Start()
	n.logger.Info().Msg("Weather notifier started")
}

// Stop cancels all scheduled jobs and waits for completion.
func (n *Notifier) Stop() {
	n.cancel()
	stopCtx := n.cron.Stop()
	<-stopCtx.Done()
	n.logger.Info().Msg("All cron jobs finished, notifier stopped")
}

// RunDue fetches due subscriptions and sends updates.
func (n *Notifier) RunDue(ctx context.Context, frequency string) {
	start := time.Now()
	n.logger.Debug().Str("frequency", frequency).Msg("starting RunDue")

	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	// RED: count this run
	n.m.CronRuns.WithLabelValues(frequency).Inc()

	// Fetch due subscriptions
	subs, err := n.repo.GetConfirmedByFrequency(ctx, frequency)
	if err != nil {
		n.logger.Error().Err(err).
			Str("frequency", frequency).
			Msg("error fetching due subscriptions")
		n.m.TechnicalErrors.WithLabelValues("fetch_due_subs", "critical").Inc()
		return
	}
	n.logger.Info().Str("frequency", frequency).Int("count", len(subs)).Msg("fetched due subscriptions")

	var wg sync.WaitGroup
	wg.Add(len(subs))

	// Send updates concurrently
	for _, sub := range subs {
		s := sub
		go func() {
			defer wg.Done()
			if err := n.SendOne(ctx, s); err != nil {
				n.logger.Error().Err(err).
					Int("subscription_id", s.ID).
					Msg("error sending update")
				n.m.TechnicalErrors.WithLabelValues("send_one", "critical").Inc()
			}
		}()
	}

	wg.Wait()

	// Observe duration
	dur := time.Since(start)
	n.m.CronRunDuration.WithLabelValues(frequency).Observe(dur.Seconds())
	n.logger.Info().Str("frequency", frequency).Dur("duration", dur).Msg("completed RunDue")
}

// SendOne obtains forecast and emails a single subscriber, then updates last_sent.
func (n *Notifier) SendOne(ctx context.Context, sub models.Subscription) error {
	start := time.Now()
	n.logger.Debug().Int("subscription_id", sub.ID).Str("city", sub.City).Msg("SendOne start")

	// Fetch weather
	forecast, err := n.weatherService.GetByCity(ctx, sub.City)
	if err != nil {
		n.logger.Error().Err(err).
			Int("subscription_id", sub.ID).
			Msg("weather fetch error")
		n.m.TechnicalErrors.WithLabelValues("weather_fetch_error", "critical").Inc()
		return err
	}

	// Send email
	if err := n.emailService.SendWeather(ctx, sub.Email, forecast); err != nil {
		n.logger.Error().Err(err).
			Str("email", sub.Email).
			Msg("email send error")
		n.m.TechnicalErrors.WithLabelValues("email_send_error", "critical").Inc()
		return err
	}

	// Update last_sent
	if err := n.repo.UpdateLastSent(ctx, sub.ID); err != nil {
		n.logger.Error().Err(err).
			Int("subscription_id", sub.ID).
			Msg("failed to update last_sent")
		n.m.TechnicalErrors.WithLabelValues("db_update_error", "critical").Inc()
		return err
	}

	dur := time.Since(start)
	n.logger.Info().Int("subscription_id", sub.ID).
		Dur("duration", dur).
		Msg("SendOne completed successfully")
	return nil
}
