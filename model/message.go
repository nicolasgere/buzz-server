package model

type Message struct {
	Id string `json:"id"`
	Topic string `json:"topic"`
	Channel string `json:"channel"`
	Data string `json:"data"`
}

type Subscribe struct {
	Topic string `json:"topic"`
	Channel string `json:"channel"`
}
