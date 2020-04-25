package model

import (
	"github.com/segmentio/fasthash/fnv1a"
	"strconv"
)

type TargetV2 struct {
	Channel string `json:"channel"`
	Topic   string `json:"topic"`
}

type MessageV2 struct {
	Type    string      `json:"type"`
	Target  TargetV2    `json:"target"`
	Payload interface{} `json:"payload"`
	Key     string      `json:"key"`
	client  *Client
}

func (self *MessageV2) GetRowKey() (r string) {
	rInt := fnv1a.HashString32(self.Target.Channel + self.Target.Topic)
	r = strconv.Itoa(int(rInt))
	return r
}

type Subscribe struct {
	Topic   string `json:"topic"`
	Channel string `json:"channel"`
}

func (self *Subscribe) GetRowKey() (r string) {
	rInt := fnv1a.HashString32(self.Channel + self.Topic)
	r = strconv.Itoa(int(rInt))
	return r
}
