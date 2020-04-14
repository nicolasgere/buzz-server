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

type Heartbeat struct {
	Topic string `json:"topic"`
	Channel string `json:"channel"`
	Key string `json:"key"`
	Data string `json:"data"`
}

type PresenceResponse struct {
	Data []string `json:"data"`
}

