package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "Create a TCP server and binds to an address/port received from options",
	Inports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OPTIONS",
			Type:        "string",
			Description: "Configures the Websocket server with address to bind to (i.e. 0.0.0.0:5000)",
			Required:    true,
		},
		registry.EntryPort{
			Name:        "IN",
			Type:        "json",
			Description: "Input port for receiving IPs and forwarding them to the corresponding connection",
			Required:    true,
		},
	},
	Outports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OUT",
			Type:        "json",
			Description: "Output port for sending IPs with data received from a specific connection",
			Required:    true,
		},
	},
}
