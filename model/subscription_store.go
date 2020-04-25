package model

import (
	"fmt"
)

type GetSubscriptionQuery struct {
	id       string
	response chan ([]string)
}

type SubscriptionQuery struct {
	idClient        string
	rowSubscription string
}

type SubscriptionStore struct {
	subscriptions map[string]map[string]bool
	clients       map[string][]string
	addChan       chan (SubscriptionQuery)
	deleteChan    chan (SubscriptionQuery)
	getChan       chan (GetSubscriptionQuery)
}

func (self *SubscriptionStore) Init() {
	self.addChan = make(chan (SubscriptionQuery))
	self.deleteChan = make(chan (SubscriptionQuery))
	self.getChan = make(chan GetSubscriptionQuery)
	self.subscriptions = map[string]map[string]bool{}
	self.clients = map[string][]string{}
	go self.Runner()
}

func (self *SubscriptionStore) Runner() {
	for {
		select {
		case query := <-self.deleteChan:
			fmt.Printf("%s:unregister \n", query.idClient)
			val, ok := self.subscriptions[query.rowSubscription]
			if ok {
				delete(val, query.idClient)
			}
		case query := <-self.addChan:
			fmt.Printf("%s:register \n", query.idClient)
			val, ok := self.subscriptions[query.rowSubscription]
			if ok {
				val[query.idClient] = true
				self.clients[query.idClient] = append(self.clients[query.idClient], query.rowSubscription)
			} else {
				self.subscriptions[query.rowSubscription] = map[string]bool{
					query.idClient: true,
				}
			}
		case query := <-self.getChan:
			fmt.Printf("%s:get \n", query.id)
			val, ok := self.subscriptions[query.id]
			if !ok {
				query.response <- []string{}
			} else {
				resp := make([]string, len(val))
				i := 0
				for key, _ := range val {
					resp[i] = key
					i++
				}
				query.response <- resp
			}

		}
	}
}

func (self *SubscriptionStore) Add(rowSubscription string, idClient string) {
	self.addChan <- SubscriptionQuery{
		idClient:        idClient,
		rowSubscription: rowSubscription,
	}
}
func (self *SubscriptionStore) Delete(rowSubscription string, idClient string) {
	self.deleteChan <- SubscriptionQuery{
		idClient:        idClient,
		rowSubscription: rowSubscription,
	}
}
func (self *SubscriptionStore) Get(id string) []string {
	query := GetSubscriptionQuery{
		id:       id,
		response: make(chan []string, 1),
	}
	self.getChan <- query
	c := <-query.response
	close(query.response)
	return c
}
