package model

import socketio "github.com/googollee/go-socket.io"

type  Appartement struct {
	Id string
	Queue string
	Rooms []Room
	Server *socketio.Server
}