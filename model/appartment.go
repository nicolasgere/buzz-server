package model

import (
	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"fmt"
	"github.com/patrickmn/go-cache"
	"github.com/segmentio/ksuid"
	"nhooyr.io/websocket"
	"time"
)

var columnFamilyName = "apt"

type Appartement struct {
	Id            string
	Queue         string
	table         *bigtable.Table
	tablePresence *bigtable.Table
	cache         *cache.Cache
	hub           *Hub
	clients       map[string]*Client
	subscriptions map[string]map[string]*Client
	pubsubClient  *pubsub.Client
	incoming      chan *MessageV2
	direct        chan *MessageV2

	unregister chan string
	register   chan *Client
}

func (self *Appartement) Init() {
	self.InitBigTable()
	self.InitPubsub()
	self.hub = newHub()
	self.clients = map[string]*Client{}
	self.subscriptions = map[string]map[string]*Client{}
	self.unregister = make(chan string)
	self.register = make(chan *Client)
	self.incoming = make(chan *MessageV2)
	self.direct = make(chan *MessageV2)
	self.cache = cache.New(5*time.Second, 30*time.Second)
	go self.ClientRunner()
	go self.ReceiveMessage()
}

func (self *Appartement) InitBigTable() {
	ctx := context.Background()
	var err error
	bigtable, err := bigtable.NewClient(ctx, "my-project-id", "my-instance")
	if err != nil {
		panic("Could not create data operations client: " + err.Error())
	}
	self.table = bigtable.Open("buzz")

	self.tablePresence = bigtable.Open("heartbeat")
}

func (self *Appartement) InitPubsub() {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, "my-project-id")

	if err != nil {
		panic(err)
	}
	self.pubsubClient = client

	topic := self.pubsubClient.Topic(self.Queue)
	exists, err := topic.Exists(ctx)
	if err != nil {
		panic(err)
	}
	if !exists {
		fmt.Printf("Topic %v doesn't exist - creating it", self.Queue)
		_, err = client.CreateTopic(ctx, self.Queue)
		if err != nil {
			panic(err)
		}
	}

	// Create the subscription if it doesn't exist.
	subscription := self.pubsubClient.Subscription(self.Queue)
	exists, err = subscription.Exists(ctx)
	if err != nil {
		panic(err)
	}
	if !exists {
		subscription, err = self.pubsubClient.CreateSubscription(context.Background(), self.Queue, pubsub.SubscriptionConfig{Topic: topic})
		if err != nil {
			panic(err)
		}
	}
	go subscription.Receive(context.Background(), func(ctx context.Context, message *pubsub.Message) {
		var m MessageV2
		message.Ack()
		err := json.Unmarshal(message.Data, &m)
		if err != nil {
			fmt.Println(err)
			message.Ack()
			return
		}
		m.Type = "message"
		fmt.Printf("pubsub:%s:%s \n", m.Target.Channel, m.Target.Topic)
		self.incoming <- &m
	})
	go func() {
		for {
			m := <-self.direct
			self.incoming <- m
		}
	}()
}

func (self *Appartement) RegisterRoom(m Subscribe, clientId string) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	key := m.GetRowKey()
	mut := bigtable.NewMutation()
	t := time.Now().Add(5 * time.Minute)
	timestamp := bigtable.Time(t)
	mut.Set(columnFamilyName, self.Queue, timestamp, []byte(clientId))
	if err := self.table.Apply(ctx, key, mut); err != nil {
		fmt.Errorf("Apply: %v", err)
	}
	return
}

func (self *Appartement) RegisterClient(conn *websocket.Conn, ctx context.Context) (err error) {
	client := &Client{
		id:            ksuid.New().String(),
		unregister:    self.unregister,
		receive:       self.incoming,
		conn:          conn,
		send:          make(chan []byte),
		lastPing:      time.Now(),
		subscriptions: map[string]bool{},
	}
	self.register <- client
	chanErr := make(chan error)
	go client.readPump(ctx, chanErr)
	go client.writePump(ctx, chanErr)
	go client.ping(ctx, chanErr)

	fmt.Printf("%s:client:connected \n", client.id)
	err = <-chanErr
	self.unregister <- client.id
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
		fmt.Printf("%s:client:close:normal \n", client.id)
		return
	}
	if err != nil {
		fmt.Printf("%s:client:close:error %v \n", client.id, err.Error())
		return
	}
	return
}

func (self *Appartement) ClientRunner() {

	for {
		select {
		case id := <-self.unregister:
			fmt.Printf("%s:unregister \n", id)
			delete(self.clients, id)
		case client := <-self.register:
			fmt.Printf("%s:register \n", client.id)
			self.clients[client.id] = client
		}
	}
}

func (self *Appartement) ReceiveMessage() {
	ctxBackground := context.Background()

	for {
		select {
		case message := <-self.incoming:

			switch message.Type {
			case "subscribe":
				{
					if message.Target.Topic == "" || message.Target.Channel == "" {
						break
					}
					fmt.Printf("message:subscribe:%s:%s %s \n", message.Target.Channel, message.Target.Topic, message.client.id)
					message.client.subscriptions[message.GetRowKey()] = true
					val, ok := self.subscriptions[message.GetRowKey()]
					if ok {
						val[message.client.id] = message.client
					} else {
						self.subscriptions[message.GetRowKey()] = map[string]*Client{}
						self.subscriptions[message.GetRowKey()][message.client.id] = message.client
					}
					self.RegisterRoom(Subscribe{
						Topic:   message.Target.Topic,
						Channel: message.Target.Channel,
					}, self.Queue)

				}
			case "presence":
				{
					ctx, _ := context.WithTimeout(ctxBackground, time.Second*2)
					fmt.Printf("message:presence:%s:%s %s \n", message.Target.Channel, message.Target.Topic, message.client.id)
					var data []string
					key := message.GetRowKey()
					v, exist := self.cache.Get(key)
					if exist {
						data = v.([]string)
					} else {
						var row bigtable.Row
						row, _ = self.tablePresence.ReadRow(ctx, key,
							bigtable.RowFilter(
								bigtable.ChainFilters(
									bigtable.TimestampRangeFilter(time.Now(), time.Now().Add(time.Second*20)),
									bigtable.LatestNFilter(1))),
						)
						i := 0
						for _, r := range row {
							data = make([]string, len(r))
							for y, x := range r {
								data[y] = string(x.Value)
							}
							i++
						}
						self.cache.Set(key, data, time.Second*5)
					}
					x, _ := json.Marshal(data)
					b, err := json.Marshal(MessageV2{
						Type:    "presence",
						Target:  message.Target,
						Payload: string(x),
					})
					if err != nil {
						fmt.Println(err)
					}

					val, ok := self.clients[message.client.id]
					if ok {
						val.send <- b
					} else {
						fmt.Printf("error:client unknow")
					}

				}
			case "heartbeat":
				{
					ctx, _ := context.WithTimeout(ctxBackground, time.Second*2)
					fmt.Printf("message:heartbeat:%s:%s %s \n", message.Target.Channel, message.Target.Topic, message.client.id)
					key := message.GetRowKey()
					columnFamilyName := "beat"
					mut := bigtable.NewMutation()
					t := time.Now().Add(15 * time.Second)
					timestamp := bigtable.Time(t)
					mut.Set(columnFamilyName, "heartbeat"+message.Key, timestamp, []byte(message.Payload))
					if err := self.tablePresence.Apply(ctx, key, mut); err != nil {
						fmt.Println(err)
					}
				}
			case "unsubscribe":
				{
					fmt.Printf("message:heartbeat:%s:%s %s \n", message.Target.Channel, message.Target.Topic, message.client.id)
					delete(message.client.subscriptions, message.GetRowKey())
					delete(self.subscriptions[message.GetRowKey()], message.client.id)
				}
			case "message":
				{
					fmt.Printf("message:new:%s:%s %s \n", message.Target.Channel, message.Target.Topic, message.Payload)
					b, err := json.Marshal(message)
					if err != nil {
						fmt.Println(err.Error())
					}
					for _, e := range self.subscriptions[message.GetRowKey()] {
						e.send <- b
					}
				}
			case "broadcast":
				{
					ctx, _ := context.WithTimeout(ctxBackground, time.Second*2)
					fmt.Printf("message:broadcast:%s:%s %s \n", message.Target.Channel, message.Target.Topic, message.client.id)
					key := message.GetRowKey()
					row, err := self.table.ReadRow(ctx, key, bigtable.RowFilter(
						bigtable.ChainFilters(
							bigtable.TimestampRangeFilter(time.Now(), time.Now().Add(time.Minute*20)),
							bigtable.LatestNFilter(1))))
					if err != nil {

					}
					for _, apt := range row {
						for _, v := range apt {
							queue := string(v.Value)
							if queue == self.Queue {
								m := *message
								m.Type = "message"
								self.direct <- &m
							} else {
								d, _ := json.Marshal(message)
								topic := self.pubsubClient.Topic(queue)
								topic.Publish(ctx, &pubsub.Message{
									Data: d,
								})
							}
						}
					}
				}
			}

		}
	}
}
