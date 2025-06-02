package model

type Subscription struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	City         string `json:"city"`
	Token        string `json:"token"`
	Confirmed    bool   `json:"confirmed"`
	Unsubscribed bool   `json:"unsubscribed"`
	CreatedAt    string `json:"created_at"`
}
