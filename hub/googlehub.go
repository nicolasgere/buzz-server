package hub

import (
	"buzz/model"
	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"fmt"
	"github.com/patrickmn/go-cache"
	"time"
)

var data = map[string] *model.Appartement{}
type GoogleHub struct {
	pubsub *pubsub.Client
	bigtable *bigtable.Client
	table *bigtable.Table
	tableHeart *bigtable.Table
	cache *cache.Cache

}
func (self *GoogleHub) RegisterApt(apt model.Appartement){
	self.cache = cache.New(5*time.Second, 30*time.Second)

	key := apt.Id
	_, ok := data[key]
	if(!ok){
		data[key] = &apt
	}
	ctx := context.Background()
	var err error
	self.bigtable, err = bigtable.NewClient(ctx, "my-project-id", "my-instance")
	if err != nil {
		panic("Could not create data operations client: "+ err.Error())
	}
	self.table = self.bigtable.Open("buzz")
	self.tableHeart = self.bigtable.Open("heartbeat")
	client, err := pubsub.NewClient(ctx, "my-project-id")
	if err != nil {
		panic(err)
	}
	self.pubsub = client
	topic := self.pubsub.Topic(apt.Queue)

	// Create the topic if it doesn't exist.
	exists, err := topic.Exists(ctx)
	if err != nil {
		panic(err)
	}
	if !exists {
		fmt.Printf("Topic %v doesn't exist - creating it", apt.Queue)
		_, err = client.CreateTopic(ctx, apt.Queue)
		if err != nil {
			panic(err)
		}
	}

	// Create the subscription if it doesn't exist.
	subscription := self.pubsub.Subscription(apt.Queue)
	exists, err = subscription.Exists(ctx)
	if err != nil {
		panic(err)
	}
	if !exists {
		subscription, err = self.pubsub.CreateSubscription(context.Background(), apt.Queue, pubsub.SubscriptionConfig{Topic: topic})
		if err != nil {
			panic(err)
		}
	}

	go subscription.Receive(context.Background(), func(ctx context.Context, message *pubsub.Message) {
		var m model.Message

		err := json.Unmarshal(message.Data, &m)
		if(err != nil){
			fmt.Println(err)
			message.Ack()
			return
		}
		d, _ := json.Marshal(m)
		fmt.Printf("message:%s:receive \n", m.Id)
		apt.Server.BroadcastToRoom("/", m.Channel + m.Topic, "entry-message", d)
		message.Ack()
	})

}

func (self *GoogleHub) RegisterRoom(apt model.Appartement, channel string, topic string)(err error){
	ctx := context.Background()
	columnFamilyName := "apt"

	key:= channel + topic
	row, err := self.table.ReadRow(ctx,key)
	if(err!= nil){
		fmt.Println(err)
		return
	}
	if row == nil {
		mut := bigtable.NewMutation()
		timestamp := bigtable.Now()
		mut.Set(columnFamilyName,apt.Queue , timestamp, []byte(apt.Queue))
		if err := self.table.Apply(ctx, key, mut); err != nil {
			return fmt.Errorf("Apply: %v", err)
		}
	} else {
		found := false
		for _, col := range row {
			if(found){
				break
			}
			for _,v := range col {
				if(string(v.Value) == apt.Queue){
				found = true
				break
			}
			}
		}
		if(found == false){
			mut := bigtable.NewMutation()
			timestamp := bigtable.Now()
			mut.Set(columnFamilyName,apt.Queue , timestamp, []byte(apt.Queue))
			if err := self.table.Apply(ctx, key, mut); err != nil {
				return fmt.Errorf("Apply: %v", err)
			}
		}
	}
	return
}

func (self *GoogleHub) UnregisterRoom(apt model.Appartement, channel string, topic string)(err error){
	ctx := context.Background()
	mut := bigtable.NewMutation()
	key := channel + topic
	mut.DeleteCellsInColumn("apt", apt.Queue)
	if err := self.table.Apply(ctx, key, mut); err != nil {
		return fmt.Errorf("Apply: %v", err)
	}
	return
}
func (self *GoogleHub) UnregisterApt(apt model.Appartement){
	key := apt.Id
	_, ok := data[key]
	if(!ok){
		delete(data,key)
	}
	return
}

func (self *GoogleHub ) BroadcastToApt(channel string, topic string, message model.Message){
	ctx := context.Background()
	key := channel + topic
	row, err := self.table.ReadRow(ctx,key )
	if(err!= nil){
	}
	d, _ := json.Marshal(message)

	for _,apt := range row {
		for _, v := range apt{
			queue := string(v.Value)
			topic := self.pubsub.Topic(queue)
			topic.Publish(ctx, &pubsub.Message{
				Data: d,
			})
		}
	}
}

func (self *GoogleHub ) Newbeat( message model.Heartbeat){
	ctx := context.Background()
	key := message.Channel + message.Topic
	columnFamilyName := "beat"
	d, _ := json.Marshal(message)
	mut := bigtable.NewMutation()
	t := time.Now().Add(15 * time.Second)
	timestamp := bigtable.Time(t)
	mut.Set(columnFamilyName,"heartbeat" + message.Key , timestamp, d)
	if err := self.tableHeart.Apply(ctx, key, mut); err != nil {
		fmt.Println(err)
	}
}


func (self *GoogleHub ) Getbeat (channel string, topic string)(err error, data []string){
	ctx := context.Background()
	key := channel + topic
	v, exist := self.cache.Get(key)
	if exist{
		data = v.([]string)
	} else {
		var row bigtable.Row
		row, err = self.tableHeart.ReadRow(ctx,key,
			bigtable.RowFilter(
				bigtable.ChainFilters(
					bigtable.TimestampRangeFilter(time.Now(), time.Now().Add(time.Second * 20)),
					bigtable.LatestNFilter(1))),
		)
		i := 0
		for _,r := range row {
			data = make([]string, len(r))
			for y,x := range r {
				data[y] = string(x.Value)
			}
			i++
		}
		self.cache.Set(key,data, time.Second *5)
	}

	return
}