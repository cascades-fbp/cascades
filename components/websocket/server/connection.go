package main

import (
	"cascades/components/websocket/utils"
	"code.google.com/p/go.net/websocket"
	uuid "github.com/nu7hatch/gouuid"
	"log"
	"time"
)

// Connection data structure
type Connection struct {
	ID      string
	WS      *websocket.Conn
	Send    chan interface{}
	Created time.Time
	Updated time.Time
}

// Connection constructor
func NewConnection(ws *websocket.Conn) *Connection {
	id, _ := uuid.NewV4()
	now := time.Now()
	conn := &Connection{
		ID:      id.String(),
		WS:      ws,
		Send:    make(chan interface{}, 256),
		Created: now,
		Updated: now,
	}
	return conn
}

// Closes the connection
func (c *Connection) Close() {
	log.Println("Connection.Close()")
	c.WS.Close()
}

// Reads from connection
func (c *Connection) Reader() {
	for {
		var payload interface{}

		// Read & decode message from connection
		err := websocket.JSON.Receive(c.WS, &payload)
		if err != nil {
			log.Printf("Connection.Reader(%v): Unhandled error: %#v", c.ID, err.Error())
		}

		// Pass data to the hub (in the proper channel)
		log.Printf("Connection.Reader(%v): Received %#v\n", c.ID, payload)
		DefaultHub.Incoming <- utils.Message{c.ID, payload}
	}
	c.Close()
}

// Write to connection
func (c *Connection) Writer() {
	for payload := range c.Send {
		log.Printf("Connection.Writer(%v): %#v\n", c.ID, payload)
		err := websocket.JSON.Send(c.WS, payload)
		if err != nil {
			log.Printf("Connection.Writer(%v): Unhandled error: %#v", c.ID, err.Error())
		}
	}
	c.WS.Close()
}
