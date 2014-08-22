package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "Create a HTTP server and binds to an address/port received from options",
	Inports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OPTIONS",
			Type:        "string",
			Description: "Configuration port to pass IP with TCP endpoint in the format i.e. 127.0.0.1:8080",
			Required:    true,
		},
		registry.EntryPort{
			Name:        "IN",
			Type:        "json",
			Description: "Input port for receiving responses in predefined JSON format",
			Required:    true,
		},
	},
	Outports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OUT",
			Type:        "json",
			Description: "Output port for emitting requests in predefined JSON format",
			Required:    true,
		},
	},
}
