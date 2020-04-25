package main

import (
	"buzz/core"
	"buzz/migration"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/segmentio/ksuid"
	"io/ioutil"
	"log"
	"net/http"
	"nhooyr.io/websocket"
	"os"
)

//var upgrader = websocket.Upgrader{
//	ReadBufferSize:  1024,
//	WriteBufferSize: 1024,
//}

func main() {
	migration.Migrate()
	id := ksuid.New()
	apt := core.Appartement{
		Id:    "apt-" + id.String(),
		Queue: "apt-" + id.String(),
	}
	apt.Init()

	router := mux.NewRouter()
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{
				"*",
				"localhost:3000",
			},
			CompressionMode:      0,
			CompressionThreshold: 0,
		})
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		//defer conn.Close(websocket.StatusInternalError, "the sky is falling")
		err = apt.RegisterClient(conn, r.Context())
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
