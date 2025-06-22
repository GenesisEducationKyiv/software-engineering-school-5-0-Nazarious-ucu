package repository

import (
	"database/sql"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

const dayHours = 24

type Subscription struct {
	ID         int
	Email      string
	City       string
	Frequency  string
	LastSentAt *time.Time
}

type SubscriptionRepository struct {
	logger *log.Logger
	DB     *sql.DB
}

func NewSubscriptionRepository(db *sql.DB, logger *log.Logger) *SubscriptionRepository {
	return &SubscriptionRepository{DB: db, logger: logger}
}

func (r *SubscriptionRepository) Create(email, city, token string, frequency string) error {
	_, err := r.DB.Exec(
		`INSERT INTO subscriptions 
    				(email, city, token, confirmed, unsubscribed, created_at, frequency, last_sent)
         VALUES (?, ?, ?, 0, 0, ?, ?, null)`,
		email, city, token, time.Now(), frequency,
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

func (r *SubscriptionRepository) GetConfirmed() ([]Subscription, error) {
	rows, err := r.DB.Query(`
		SELECT id, email, city, frequency, last_sent
		FROM subscriptions
		WHERE confirmed = 1 AND unsubscribed = 0
	`)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			r.logger.Println(err)
		}
	}(rows)

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
				shouldSend = now.Sub(*sub.LastSentAt) >= dayHours*time.Hour
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

func (r *SubscriptionRepository) GetConfirmedByFrequency(frequency string) ([]Subscription, error) {
	rows, err := r.DB.Query(`
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

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
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
