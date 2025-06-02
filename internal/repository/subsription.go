package repository

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"time"
)

type Subscription struct {
	ID         int
	Email      string
	City       string
	Frequency  string
	LastSentAt *time.Time
}

type SubscriptionRepository struct {
	DB *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{DB: db}
}

func (r *SubscriptionRepository) Create(email, city, token string, frequency string) error {
	_, err := r.DB.Exec(
		`INSERT INTO subscriptions (email, city, token, confirmed, unsubscribed, created_at, frequency, last_sent)
         VALUES (?, ?, ?, 0, 0, ?, ?, null)`,
		email, city, token, time.Now(),
	)
	return err
}

func (r *SubscriptionRepository) Confirm(token string) (bool, error) {
	res, err := r.DB.Exec(
		"UPDATE subscriptions SET confirmed = 1 WHERE token = ?", token,
	)
	if err != nil {
		return false, err
	}
	count, err := res.RowsAffected()
	return count > 0, err
}

func (r *SubscriptionRepository) Unsubscribe(token string) (bool, error) {
	res, err := r.DB.Exec(
		"UPDATE subscriptions SET unsubscribed = 1 WHERE token = ?", token,
	)
	if err != nil {
		return false, err
	}
	count, err := res.RowsAffected()
	return count > 0, err
}

func (r *SubscriptionRepository) GetConfirmedSubscriptions() ([]Subscription, error) {
	rows, err := r.DB.Query(`
		SELECT id, email, city, frequency, last_sent
		FROM subscriptions
		WHERE confirmed = 1 AND unsubscribed = 0
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	now := time.Now()

	for rows.Next() {
		var sub Subscription
		var lastSent sql.NullTime

		if err := rows.Scan(&sub.ID, &sub.Email, &sub.City, &sub.Frequency, &lastSent); err != nil {
			return nil, err
		}

		if lastSent.Valid {
			sub.LastSentAt = &lastSent.Time
		}

		shouldSend := false
		if sub.LastSentAt == nil {
			shouldSend = true
		} else {
			switch sub.Frequency {
			case "hourly":
				shouldSend = now.Sub(*sub.LastSentAt) >= time.Hour
			case "daily":
				shouldSend = now.Sub(*sub.LastSentAt) >= 24*time.Hour
			}
		}

		if shouldSend {
			subs = append(subs, sub)
		}
	}

	return subs, rows.Err()
}

func (r *SubscriptionRepository) UpdateLastSent(subscriptionID int) error {
	_, err := r.DB.Exec(
		"UPDATE subscriptions SET last_sent = ? WHERE id = ?",
		time.Now(), subscriptionID,
	)
	return err
}
