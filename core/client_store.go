package core

import (
	"fmt"
)

type GetClientQuery struct {
	id       string
	response chan (*Client)
}

type ClientStore struct {
	clients    map[string]*Client
	addChan    chan (*Client)
	deleteChan chan (string)
	getChan    chan (GetClientQuery)
}

func (self *ClientStore) Init() {
	self.addChan = make(chan (*Client))
	self.deleteChan = make(chan (string))
	self.getChan = make(chan GetClientQuery)
	self.clients = map[string]*Client{}
	go self.Runner()
}

func (self *ClientStore) Runner() {
	for {
		select {
		case id := <-self.deleteChan:
			fmt.Printf("%s:unregister \n", id)
			delete(self.clients, id)
		case client := <-self.addChan:
			fmt.Printf("%s:register \n", client.id)
			self.clients[client.id] = client
		case query := <-self.getChan:
			fmt.Printf("%s:get \n", query.id)
			val, ok := self.clients[query.id]
			if ok {
				query.response <- val
			} else {
				query.response <- nil
			}

		}
	}
}

func (self *ClientStore) Add(client *Client) {
	self.addChan <- client
}
func (self *ClientStore) Delete(id string) {
	self.deleteChan <- id
}
func (self *ClientStore) Get(id string) *Client {
	query := GetClientQuery{
		id:       id,
		response: make(chan *Client, 1),
	}
	self.getChan <- query
	c := <-query.response
	close(query.response)
	return c
}
