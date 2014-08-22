package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "Create a TCP server and binds to an address/port received from options",
	Inports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OPTIONS",
			Type:        "all",
			Description: "Configuration port to pass IP with TCP endpoint in the format tcp://x.x.x.x:y",
			Required:    true,
		},
		registry.EntryPort{
			Name:        "IN",
			Type:        "all",
			Description: "Input port for receiving IPs",
			Required:    true,
		},
	},
	Outports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OUT",
			Type:        "all",
			Description: "Output port for sending IPs",
			Required:    true,
		},
	},
}
