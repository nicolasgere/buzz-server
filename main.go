package main

import (
	"buzz/migration"
	"buzz/model"
	"fmt"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"github.com/segmentio/ksuid"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	migration.Migrate()

	id := ksuid.New()
	apt := model.Appartement{
		Id:    "apt-" + id.String(),
		Queue: "apt-" + id.String(),
	}
	apt.Init()

	router := mux.NewRouter()
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Println(err)
			return
		}
		apt.RegisterClient(conn)
	})
	router.HandleFunc("/message/{channel}/{topic}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		channel := vars["channel"]
		topic := vars["topic"]
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "can't read body", http.StatusBadRequest)
		}
		fmt.Printf("publish %s %s %s \n", channel, topic, body)
		//message := model.Message{
		//	Id:      ksuid.New().String(),
		//	Topic:   topic,
		//	Channel: channel,
		//	Data:    string(body),
		//}
		////hub.Get().BroadcastToApt(message)
		w.WriteHeader(http.StatusOK)
	})
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "PUT", "OPTIONS", "POST", "DELETE"},
		AllowCredentials: true,
	})
	handler := c.Handler(router)
	port := "8080"
	portS := os.Getenv("port")

	if portS != "" {
		port = portS
	}
	log.Println("Serving at localhost:" + port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
