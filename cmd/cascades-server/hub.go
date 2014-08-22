package main

import (
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

	// Data handling channel
	Data chan *Message
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
		case m := <-self.Data:
			log.Printf("Hub.Start: Data received: %#v", m)
			switch m.Protocol {
			case ProtocolComponent:
				switch m.Command {
				case "list":
					self.nofloComponentList(m.ConnId)
				}
			case ProtocolRuntime:
				switch m.Command {
				case "getruntime":
					self.nofloRuntimeGetruntime(m.ConnId)
				}
			}
		}
	}
}

func (self *Hub) nofloComponentList(connId string) {
	entries := componentsDb.List()
	for n, e := range entries {
		payload := ComponentPayload{
			Name:        n,
			Description: e.Description,
		}
		if e.Elementary {
			payload.Subgraph = false
		} else {
			payload.Subgraph = true
		}
		payload.Inports = []ComponentPayloadInport{}
		for _, p := range e.Inports {
			port := ComponentPayloadInport{
				Id:          p.Name,
				Type:        p.Type,
				Description: p.Description,
				Addressable: false,
				Required:    p.Required,
			}
			payload.Inports = append(payload.Inports, port)
		}
		payload.Outports = []ComponentPayloadOutport{}
		for _, p := range e.Outports {
			port := ComponentPayloadOutport{
				Id:          p.Name,
				Type:        p.Type,
				Description: p.Description,
				Addressable: false,
				Required:    p.Required,
			}
			payload.Outports = append(payload.Outports, port)
		}
		message := Message{
			Protocol: ProtocolComponent,
			Command:  "component",
			Payload:  payload,
		}
		self.connections[connId].Send <- message
	}

}

func (self *Hub) nofloRuntimeGetruntime(connId string) {
	payload := RuntimePayload{
		Type:    RuntimeType,
		Version: RuntimeProtocolVersion,
		Capabilities: []Capability{
			//CapabilityProtocolRuntime,
			CapabilityProtocolGraph,
			CapabilityProtocolComponent,
			//CapabilityProtocolNetwork,
			//CapabilityNetworkPersist,
		},
	}
	message := Message{
		Protocol: ProtocolRuntime,
		Command:  "runtime",
		Payload:  payload,
	}
	self.connections[connId].Send <- message
}

//
// The default Hub instance used all around this code
//
var DefaultHub = Hub{
	connections: make(map[string]*Connection),
	Register:    make(chan *Connection),
	Unregister:  make(chan *Connection),
	Data:        make(chan *Message),
}

//
// Init the package defaults
//
func init() {
	ID, _ := uuid.NewV4()
	DefaultHub.ID = ID.String()
}
