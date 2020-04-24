package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"nhooyr.io/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 1000 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 20 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 10000
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	id string
	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Buffered channel of inbound messages.
	receive chan *MessageV2

	unregister chan string

	lastPing time.Time

	subscriptions map[string]bool
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump(ctx context.Context, chanErr chan error) {
	c.conn.SetReadLimit(maxMessageSize)
	var err error
	for {
		var d io.Reader
		_, d, err = c.conn.Reader(ctx)
		if err != nil {
			break
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(d)
		data1 := bytes.TrimSpace(bytes.Replace(buf.Bytes(), newline, space, -1))
		var message MessageV2
		errMarshal := json.Unmarshal(data1, &message)
		if errMarshal != nil {
			fmt.Println(err.Error())
		}
		message.client = c
		c.receive <- &message
	}
	if err != nil {
		chanErr <- err
	}
}

func (c *Client) ping(ctx context.Context, chanErr chan error) {
	ticker := time.Tick(20 * time.Second)
	var err error
L:
	for {
		select {
		case <-ticker:
			ctx, _ := context.WithTimeout(context.Background(), time.Second*15)

			err = c.conn.Ping(ctx)
			if err == nil {
				fmt.Printf("%s:client:ping:ok \n", c.id)
			} else {
				fmt.Printf("%s:client:ping:error \n", c.id)
				break L
			}
		case <-ctx.Done():
			break L
		}

	}
	if err != nil {
		chanErr <- err
	}

}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump(ctx context.Context, chanErr chan error) {
	var err error
L:
	for {
		select {
		case message := <-c.send:
			err = c.conn.Write(ctx, websocket.MessageText, message)
			if err != nil {
				break L

			}
		case <-ctx.Done():
			break L
		}
	}
	if err != nil {
		chanErr <- err
	}
}

// serveWs handles websocket requests from the peer.
