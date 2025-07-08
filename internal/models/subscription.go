package models

import "time"

type Subscription struct {
	ID         int
	Email      string
	City       string
	Frequency  string
	LastSentAt *time.Time
}

type UserSubData struct {
	Email     string `json:"email" binding:"required,email"`
	City      string `json:"city" binding:"required"`
	Frequency string `json:"frequency" binding:"required,oneof=hourly daily"`
}
