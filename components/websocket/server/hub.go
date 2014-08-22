package main

import (
	"cascades/components/websocket/utils"
	uuid "github.com/nu7hatch/gouuid"
	"log"
)

// The communication hub between the active connections
type Hub struct {
	// Unique name for this hub
	ID string

	// All connections
	connections map[string]*Connection

	// Register requests from the connections
	Register chan *Connection

	// Unregister requests from connections
	Unregister chan *Connection

	// Data handling channels
	Incoming chan utils.Message
	Outgoing chan utils.Message
}

// Start the hub and listen for incoming packets
func (self *Hub) Start() {
	log.Println("Hub.Start")
	for {
		select {
		case c := <-self.Register:
			log.Printf("Hub.Start: Register %v\n", c.ID)
			self.connections[c.ID] = c
			log.Printf("Hub.Start: Connections = %#v", self.connections)
		case c := <-self.Unregister:
			log.Printf("Hub.Start: Unregister %v\n", c.ID)
			delete(self.connections, c.ID)
		case msg := <-self.Outgoing:
			log.Println("Hub.Start: Outgoing data:", msg)
			if c, ok := self.connections[msg.CID]; ok {
				c.Send <- msg.Payload
			}
		}
	}
}

//
// The default Hub instance used all around this code
//
var DefaultHub = Hub{
	connections: make(map[string]*Connection),
	Register:    make(chan *Connection),
	Unregister:  make(chan *Connection),
	Incoming:    make(chan utils.Message),
	Outgoing:    make(chan utils.Message),
}

//
// Init the package defaults
//
func init() {
	ID, _ := uuid.NewV4()
	DefaultHub.ID = ID.String()
}
