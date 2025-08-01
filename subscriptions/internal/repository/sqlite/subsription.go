package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/metrics"
	"github.com/Nazarious-ucu/weather-subscription-api/subscriptions/internal/models"
	"github.com/rs/zerolog"
)

// SubscriptionRepository handles CRUD operations on subscriptions with structured logging and metrics.
type SubscriptionRepository struct {
	DB  *sql.DB
	log zerolog.Logger
	m   *metrics.Metrics
}

// NewSubscriptionRepository constructs a repository with logger context and metrics collector.
func NewSubscriptionRepository(
	db *sql.DB,
	logger zerolog.Logger,
	m *metrics.Metrics,
) *SubscriptionRepository {
	logger = logger.With().Str("component", "SubscriptionRepository").Logger()
	return &SubscriptionRepository{DB: db, log: logger, m: m}
}

// Create inserts a new subscription, returns ErrSubscriptionExists if duplicate.
func (r *SubscriptionRepository) Create(
	ctx context.Context,
	data models.UserSubData,
	token string,
) error {
	start := time.Now()
	r.log.Debug().Ctx(ctx).
		Str("email", data.Email).
		Str("city", data.City).
		Msg("checking existing subscription count")

	var cnt int
	err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM subscriptions WHERE email = ? AND city = ?`,
		data.Email, data.City,
	).Scan(&cnt)
	if err != nil {
		r.log.Error().Err(err).Ctx(ctx).
			Msg("failed to query subscription count")
		r.m.TechnicalErrors.WithLabelValues("db_query_error", err.Error(), "critical").Inc()
		return err
	}
	if cnt > 0 {
		r.log.Warn().Ctx(ctx).
			Str("email", data.Email).
			Str("city", data.City).
			Msg("subscription already exists, abort create")
		r.m.BusinessErrors.WithLabelValues("subscription_exists", "409", "warning").Inc()
		return errors.New("subscription already exists")
	}

	r.log.Info().Ctx(ctx).
		Str("email", data.Email).
		Str("city", data.City).
		Msg("inserting new subscription record")

	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO subscriptions 
		    (email, city, token, confirmed, unsubscribed, created_at, frequency, last_sent)
		 VALUES (?, ?, ?, 0, 0, ?, ?, null)`,
		data.Email, data.City, token, time.Now(), data.Frequency,
	)
	dur := time.Since(start)
	if err != nil {
		r.log.Error().Err(err).Ctx(ctx).
			Dur("duration", dur).
			Msg("failed to insert subscription")
		r.m.TechnicalErrors.WithLabelValues("db_insert_error", err.Error(), "critical").Inc()
		return err
	}

	r.log.Info().Ctx(ctx).
		Str("email", data.Email).
		Str("city", data.City).
		Dur("duration", dur).
		Msg("subscription created successfully")
	return nil
}

// Confirm marks a subscription as confirmed by token.
func (r *SubscriptionRepository) Confirm(ctx context.Context, token string) (bool, error) {
	start := time.Now()
	r.log.Debug().Ctx(ctx).Str("token", token).Msg("confirming subscription token")

	res, err := r.DB.ExecContext(ctx,
		"UPDATE subscriptions SET confirmed = 1 WHERE token = ?", token,
	)
	dur := time.Since(start)
	if err != nil {
		r.log.Error().Err(err).Ctx(ctx).
			Str("token", token).
			Msg("failed to execute confirm update")
		r.m.TechnicalErrors.WithLabelValues("db_update_error", err.Error(), "critical").Inc()
		return false, err
	}
	count, err := res.RowsAffected()
	if err != nil {
		r.log.Error().Err(err).Ctx(ctx).
			Str("token", token).
			Msg("failed to get rows affected for confirm")
		r.m.TechnicalErrors.WithLabelValues("db_rows_error", err.Error(), "critical").Inc()
		return false, err
	}

	r.log.Info().
		Str("token", token).
		Dur("duration", dur).
		Msg("subscription confirm completed")
	return count > 0, nil
}

// Unsubscribe marks a subscription as unsubscribed by token.
func (r *SubscriptionRepository) Unsubscribe(ctx context.Context, token string) (bool, error) {
	start := time.Now()
	r.log.Debug().Ctx(ctx).Str("token", token).Msg("unsubscribing subscription token")

	res, err := r.DB.ExecContext(ctx,
		"UPDATE subscriptions SET unsubscribed = 1 WHERE token = ?", token,
	)
	dur := time.Since(start)
	if err != nil {
		r.log.Error().Err(err).Ctx(ctx).
			Str("token", token).
			Msg("failed to execute unsubscribe update")
		r.m.TechnicalErrors.WithLabelValues("db_update_error", err.Error(), "critical").Inc()
		return false, err
	}
	count, err := res.RowsAffected()
	if err != nil {
		r.log.Error().Err(err).Ctx(ctx).
			Str("token", token).
			Msg("failed to get rows affected for unsubscribe")
		r.m.TechnicalErrors.WithLabelValues("db_rows_error", err.Error(), "critical").Inc()
		return false, err
	}

	r.log.Info().Ctx(ctx).
		Str("token", token).
		Dur("duration", dur).
		Msg("subscription unsubscribed successfully")
	return count > 0, nil
}

// UpdateLastSent updates the last_sent timestamp for a subscription.
func (r *SubscriptionRepository) UpdateLastSent(ctx context.Context, subscriptionID int) error {
	start := time.Now()
	r.log.Debug().Ctx(ctx).Int("subscription_id", subscriptionID).Msg("updating last_sent timestamp")

	_, err := r.DB.ExecContext(ctx,
		"UPDATE subscriptions SET last_sent = ? WHERE id = ?", time.Now(), subscriptionID,
	)
	dur := time.Since(start)
	if err != nil {
		r.log.Error().Err(err).Ctx(ctx).
			Int("subscription_id", subscriptionID).
			Msg("failed to update last_sent timestamp")
		r.m.TechnicalErrors.WithLabelValues("db_update_error", err.Error(), "critical").Inc()
		return err
	}

	r.log.Info().Ctx(ctx).
		Int("subscription_id", subscriptionID).
		Dur("duration", dur).
		Msg("last_sent timestamp updated")
	return nil
}

// GetConfirmedByFrequency retrieves all confirmed, non-unsubscribed subscriptions by frequency.
func (r *SubscriptionRepository) GetConfirmedByFrequency(
	ctx context.Context, frequency string,
) ([]models.Subscription, error) {
	start := time.Now()
	r.log.Debug().Ctx(ctx).Str("frequency", frequency).Msg("querying confirmed subscriptions by frequency")

	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, email, city, frequency, last_sent
		FROM subscriptions
		WHERE confirmed = 1 AND unsubscribed = 0 AND frequency = ?`, frequency,
	)
	dur := time.Since(start)
	if err != nil {
		r.log.Error().Err(err).Ctx(ctx).
			Str("frequency", frequency).
			Msg("failed to query subscriptions by frequency")
		r.m.TechnicalErrors.WithLabelValues("db_query_error", err.Error(), "critical").Inc()
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			r.log.Error().Err(err).Ctx(ctx).
				Str("frequency", frequency).
				Msg("failed to close rows after query")
			r.m.TechnicalErrors.WithLabelValues("db_rows_close_error", err.Error(), "critical").Inc()
		} else {
			r.log.Debug().Ctx(ctx).
				Str("frequency", frequency).
				Msg("rows closed successfully after query")
		}
	}(rows)

	var subs []models.Subscription
	for rows.Next() {
		var sub models.Subscription
		var lastSent sql.NullTime

		if err := rows.Scan(&sub.ID, &sub.Email, &sub.City, &sub.Frequency, &lastSent); err != nil {
			r.log.Error().Err(err).Ctx(ctx).
				Msg("failed to scan subscription row")
			r.m.TechnicalErrors.WithLabelValues("db_scan_error", err.Error(), "critical").Inc()
			return nil, err
		}

		if lastSent.Valid {
			sub.LastSentAt = &lastSent.Time
		}
		subs = append(subs, sub)
	}

	if err := rows.Err(); err != nil {
		r.log.Error().Err(err).Ctx(ctx).
			Msg("row iteration error")
		r.m.TechnicalErrors.WithLabelValues("db_rows_error", err.Error(), "critical").Inc()
		return nil, err
	}

	r.log.Info().Ctx(ctx).
		Str("frequency", frequency).
		Int("count", len(subs)).
		Dur("duration", dur).
		Msg("retrieved confirmed subscriptions")
	return subs, nil
}
