package core

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

type Appartement struct {
	Id             string
	Queue          string
	DbSubscription Db
	DbPresence     Db
	cache          *cache.Cache
	clients        ClientStore
	subscriptions  SubscriptionStore
	pubsubClient   *pubsub.Client
	incoming       chan *MessageV2
	direct         chan *MessageV2
}

func (self *Appartement) Init() {
	self.InitPubsub()
	self.clients = ClientStore{}
	self.clients.Init()
	self.subscriptions = SubscriptionStore{}
	self.subscriptions.Init()
	self.DbSubscription = Db{}
	self.DbSubscription.Init("buzz", "apt")
	self.DbPresence = Db{}
	self.DbPresence.Init("heartbeat", "beat")
	self.incoming = make(chan *MessageV2)
	self.direct = make(chan *MessageV2)
	self.cache = cache.New(5*time.Second, 30*time.Second)
	go self.ReceiveMessage()
	go func() {
		for {
			m := <-self.direct
			self.incoming <- m
		}
	}()
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

}

//func (self *Appartement) RegisterRoom(m Subscribe, clientId string) (err error) {
//	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
//	key := m.GetRowKey()
//	mut := bigtable.NewMutation()
//	t := time.Now().Add(5 * time.Minute)
//	timestamp := bigtable.Time(t)
//	mut.Set(columnFamilyName, self.Queue, timestamp, []byte(clientId))
//	err = self.table.Apply(ctx, key, mut)
//	return
//}

func (self *Appartement) RegisterClient(conn *websocket.Conn, ctx context.Context) (err error) {
	client := &Client{
		id:      ksuid.New().String(),
		receive: self.incoming,
		conn:    conn,
		send:    make(chan []byte),
	}
	self.clients.Add(client)
	chanErr := make(chan error)
	go client.readPump(ctx, chanErr)
	go client.writePump(ctx, chanErr)
	go client.ping(ctx, chanErr)

	fmt.Printf("%s:client:connected \n", client.id)
	err = <-chanErr
	self.clients.Delete(client.id)
	self.subscriptions.DeleteClient(client.id)

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

func (self *Appartement) ReceiveMessage() {
	ctxBackground := context.Background()

	for {
		select {
		case message := <-self.incoming:
			switch message.Type {
			case "subscribe":
				{
					if message.Target.Topic == "" || message.Target.Channel == "" {
						continue
					}
					fmt.Printf("message:subscribe:%s:%s %s \n", message.Target.Channel, message.Target.Topic, message.clientId)
					self.subscriptions.Add(message.GetRowKey(), message.clientId)
					err := self.DbSubscription.Create(message.GetRowKey(), self.Queue, self.Queue, 5*time.Minute)
					if err != nil {
						fmt.Errorf("%s:subscribe:error %v \n", message.clientId, err.Error())
					}

				}
			case "presence":
				{
					fmt.Printf("message:presence:%s:%s %s \n", message.Target.Channel, message.Target.Topic, message.clientId)
					data := []string{}
					key := message.GetRowKey()
					v, exist := self.cache.Get(key)
					if exist {
						data = v.([]string)
					} else {
						var err error
						data, err = self.DbPresence.GetRows(key,
							bigtable.RowFilter(
								bigtable.ChainFilters(
									bigtable.TimestampRangeFilter(time.Now(), time.Now().Add(time.Second*60)),
									bigtable.LatestNFilter(1))),
						)
						if err != nil {
							fmt.Errorf("%s:presence:error %v \n", message.clientId, err.Error())
							continue
						}
						self.cache.Set(key, data, time.Second*2)
					}
					x, err := json.Marshal(data)
					if err != nil {
						fmt.Errorf("%s:presence:error %v \n", message.clientId, err.Error())
						continue
					}
					b, err := json.Marshal(MessageV2{
						Type:    "presence",
						Target:  message.Target,
						Payload: string(x),
					})
					if err != nil {
						fmt.Errorf("%s:presence:error %v \n", message.clientId, err.Error())
						continue
					}

					client := self.clients.Get(message.clientId)
					if client != nil {
						client.send <- b
					} else {
						if err != nil {
							fmt.Errorf("%s:presence:client_unknown \n", message.clientId)
						}
					}

				}
			case "heartbeat":
				{
					fmt.Printf("message:heartbeat:%s:%s %s \n", message.Target.Channel, message.Target.Topic, message.clientId)
					self.DbPresence.Create(message.GetRowKey(), message.Key, fmt.Sprintf("%v", message.Payload), time.Second*15)
				}
			case "unsubscribe":
				{
					fmt.Printf("message:heartbeat:%s:%s %s \n", message.Target.Channel, message.Target.Topic, message.clientId)
					self.subscriptions.Delete(message.GetRowKey(), message.clientId)
				}
			case "message":
				{
					fmt.Printf("message:new:%s:%s \n", message.Target.Channel, message.Target.Topic)
					b, err := json.Marshal(message)
					if err != nil {
						fmt.Errorf("%s:message:error %v \n", message.clientId, err.Error())
						continue
					}
					for _, e := range self.subscriptions.Get(message.GetRowKey()) {
						c := self.clients.Get(e)
						if self.clients.Get(e) != nil {
							c.send <- b
						}
					}
				}
			case "broadcast":
				{
					ctx, _ := context.WithTimeout(ctxBackground, time.Second*2)
					fmt.Printf("message:broadcast:%s:%s %s \n", message.Target.Channel, message.Target.Topic, message.clientId)
					key := message.GetRowKey()
					rows, err := self.DbSubscription.GetRows(key, bigtable.RowFilter(
						bigtable.ChainFilters(
							bigtable.TimestampRangeFilter(time.Now(), time.Now().Add(time.Minute*20)),
							bigtable.LatestNFilter(1))))
					if err != nil {
						fmt.Errorf("%s:broadcast:error %v \n", message.clientId, err.Error())
						continue
					}
					for _, apt := range rows {
						if apt == self.Queue {
							m := *message
							m.Type = "message"
							self.direct <- &m
						} else {
							d, _ := json.Marshal(message)
							topic := self.pubsubClient.Topic(apt)
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
