package hub

import "buzz/model"

type Hub interface {
	RegisterApt(apt model.Appartement)
	UnregisterApt(apt model.Appartement)
	RegisterRoom(apt model.Appartement, channel string, topic string)(err error)
	UnregisterRoom(apt model.Appartement, channel string, topic string)(err error)
	BroadcastToApt(channel string, topic string, message model.Message)
	Newbeat(message model.Heartbeat)
	Getbeat(channel string, topic string)(err error, data []string)

}

var obj = &GoogleHub{}

func Get() Hub{
	return obj
}