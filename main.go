package main

import (
	"buzz/hub"
	"buzz/model"
	"fmt"
	engineio "github.com/googollee/go-engine.io"
	"github.com/googollee/go-engine.io/transport"
	"github.com/googollee/go-engine.io/transport/polling"
	"github.com/googollee/go-engine.io/transport/websocket"
	"github.com/googollee/go-socket.io"
	"github.com/rs/cors"
	"github.com/segmentio/ksuid"
	"log"
	"net/http"
	"os"
)

func main() {
	pt := polling.Default
	

	wt := websocket.Default
	wt.CheckOrigin = func(req *http.Request) bool {
		return true
	}

	server, err := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			pt,
			wt,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	id := ksuid.New()
	apt := model.Appartement{
		Id:    "apt-" + id.String(),
		Queue: "apt-" + id.String(),
		Server:server,
	}
	hub.Get().RegisterApt(apt)


	server.OnConnect("/", func(s socketio.Conn) error {
		fmt.Printf("client:%s:connected \n", s.ID())
		return nil
	})
	server.OnEvent("/", "subscribe", func(s socketio.Conn, msg model.Subscribe) {
		fmt.Printf("client:%s:subscribe %s %s \n", s.ID(), msg.Topic, msg.Channel)
		hub.Get().RegisterRoom(apt, msg.Channel, msg.Topic)
		s.Join(msg.Channel + msg.Topic)
	})
	server.OnEvent("/", "unsubscribe", func(s socketio.Conn, msg model.Subscribe) {
		fmt.Printf("client:%s:unsubscribe %s %s \n", s.ID(), msg.Topic, msg.Channel)
		hub.Get().UnregisterRoom(apt, msg.Channel, msg.Topic)
		s.Leave(msg.Channel + msg.Topic)
	})
	server.OnEvent("/", "message", func(s socketio.Conn, msg model.Message) string {
		fmt.Printf("client:%s:publish %s %s %s \n", s.ID(), msg.Topic, msg.Channel, msg.Data)
		msg.Id = ksuid.New().String()
		hub.Get().BroadcastToApt(msg.Channel, msg.Topic, msg )
		return "recv " + msg.Id
	})


	server.OnError("/", func(s socketio.Conn, e error) {
		fmt.Printf("client:%s:error %s \n", s.ID(), e.Error())
		s.LeaveAll()
	})
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Printf("client:%s:disconnected %s \n", s.ID(), reason)
		s.LeaveAll()
	})
	go server.Serve()
	defer server.Close()

	mux := http.NewServeMux()
	mux.Handle("/socket/", server)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:  []string{"GET", "PUT", "OPTIONS", "POST", "DELETE"},
		AllowCredentials: true,
	})
	handler := c.Handler(mux)
	port := "8080"
	portS := os.Getenv("port")

	if(portS != ""){
		port = portS
	}
	log.Println("Serving at localhost:" + port)
	log.Fatal(http.ListenAndServe(":" + port, handler))
}