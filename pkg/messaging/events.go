package messaging

type NewSubscriptionEvent struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type Weather struct {
	Temperature float64 `json:"temperature"`
	City        string  `json:"city"`
	Description string  `json:"description"`
}

type WeatherNotifyEvent struct {
	Email   string  `json:"email"`
	Weather Weather `json:"weather"`
}
