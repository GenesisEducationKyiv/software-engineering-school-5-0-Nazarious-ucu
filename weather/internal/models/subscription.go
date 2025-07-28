package models

import "time"

type Subscription struct {
	ID         int
	Email      string
	City       string
	Frequency  string
	LastSentAt *time.Time
}
