package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	uuid "github.com/nu7hatch/gouuid"
	"io"
	"log"
	"time"
)

// Connection data structure
type Connection struct {
	ID string
	WS *websocket.Conn
	//Send    chan *Message
	Send    chan interface{}
	Created time.Time
	Updated time.Time
}

// Connection constructor
func NewConnection(ws *websocket.Conn) *Connection {
	id, _ := uuid.NewV4()
	now := time.Now()
	conn := &Connection{
		ID: id.String(),
		WS: ws,
		//Send:    make(chan *Message, 256),
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
	log.Printf("Connection.Reader(%v)", c.ID)
	for {
		// Read & decode message from connection
		var message *Message
		err := websocket.JSON.Receive(c.WS, &message)
		//var message []byte
		//err := websocket.Message.Receive(c.WS, &message)

		if err == io.EOF {
			break
		} else if _, ok := err.(*json.SyntaxError); ok {
			log.Printf("Connection.Reader(%v): ERROR reading JSON from WS! Error: %#v\n", c.ID, err)
			continue
		} else if err != nil {
			log.Panicf("Connection.Reader(%v): Unhandled error: %#v", c.ID, err)
		}

		//TODO: Process the message if required
		message.ConnId = c.ID

		// Pass message to the hub (in the proper channel)
		//log.Printf("Connection.Reader: %#v", message)
		DefaultHub.Data <- message
	}
	c.Close()
}

// Write to connection
func (c *Connection) Writer() {
	log.Printf("Connection.Writer(%v)", c.ID)
	for message := range c.Send {
		//log.Printf("Connection.Writer(%v): %#v\n", c.ID, message)
		err := websocket.JSON.Send(c.WS, message)
		if err == io.EOF {
			break
		} else if _, ok := err.(*json.SyntaxError); ok {
			log.Printf("Connection.Writer(%v): ERROR sending JSON to WS! Error: %#v\n", c.ID, err)
			continue
		} else if err != nil {
			log.Printf("Connection.Writer(%v): Unhandled error: %#v", c.ID, err)
			DefaultHub.Unregister <- c
		}
	}
	c.WS.Close()
}
