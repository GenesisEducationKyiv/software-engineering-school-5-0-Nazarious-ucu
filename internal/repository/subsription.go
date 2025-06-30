package repository

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/handlers/subscription"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"

	_ "modernc.org/sqlite"
)

type SubscriptionRepository struct {
	logger *log.Logger
	DB     *sql.DB
}

func NewSubscriptionRepository(db *sql.DB, logger *log.Logger) *SubscriptionRepository {
	return &SubscriptionRepository{DB: db, logger: logger}
}

func (r *SubscriptionRepository) Create(
	ctx context.Context,
	data subscription.UserSubData,
	token string,
) error {
	var cnt int
	err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM subscriptions WHERE email = ? AND city = ?`,
		data.Email, data.City,
	).Scan(&cnt)
	if err != nil {
		return err
	}
	if cnt > 0 {
		return subscription.ErrSubscriptionExists
	}
	r.logger.Println("Creating subscription for email:", data.Email, "city:", data.City)
	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO subscriptions 
    				(email, city, token, confirmed, unsubscribed, created_at, frequency, last_sent)
         VALUES (?, ?, ?, 0, 0, ?, ?, null)`,
		data.Email, data.City, token, time.Now(), data.Frequency,
	)
	return err
}

func (r *SubscriptionRepository) Confirm(ctx context.Context, token string) (bool, error) {
	res, err := r.DB.ExecContext(ctx,
		"UPDATE subscriptions SET confirmed = 1 WHERE token = ?", token,
	)
	if err != nil {
		return false, err
	}
	count, err := res.RowsAffected()
	return count > 0, err
}

func (r *SubscriptionRepository) Unsubscribe(ctx context.Context, token string) (bool, error) {
	res, err := r.DB.ExecContext(ctx,
		"UPDATE subscriptions SET unsubscribed = 1 WHERE token = ?", token,
	)
	if err != nil {
		return false, err
	}
	count, err := res.RowsAffected()
	return count > 0, err
}

func (r *SubscriptionRepository) UpdateLastSent(subscriptionID int, ctx context.Context) error {
	_, err := r.DB.ExecContext(ctx,
		"UPDATE subscriptions SET last_sent = ? WHERE id = ?",
		time.Now(), subscriptionID,
	)
	return err
}

func (r *SubscriptionRepository) GetConfirmedByFrequency(frequency string,
	ctx context.Context,
) ([]models.Subscription, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, email, city, frequency, last_sent
		FROM subscriptions
		WHERE confirmed = 1 AND unsubscribed = 0 AND frequency = ?`, frequency,
	)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			r.logger.Println(err)
		}
	}(rows)

	var subs []models.Subscription
	for rows.Next() {
		var sub models.Subscription
		var lastSent sql.NullTime

		if err := rows.Scan(&sub.ID, &sub.Email, &sub.City, &sub.Frequency, &lastSent); err != nil {
			return nil, err
		}

		if lastSent.Valid {
			sub.LastSentAt = &lastSent.Time
		}

		subs = append(subs, sub)
	}

	return subs, rows.Err()
}
