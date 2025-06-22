package models

type ErrorResponse struct {
	Error string `json:"error"`
}

type SubscribeResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
