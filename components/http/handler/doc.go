package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: `Handles HTTP requests as substreams from http-server component in the format: ID, Method, URI. 
Emits responses in the format: ID, Status, Data.`,
	Inports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "IN",
			Type:        "json",
			Description: "Input port for receiving requests in a predefined JSON format",
			Required:    true,
		},
	},
	Outports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OUT",
			Type:        "substream",
			Description: "Output port for emitting responses in a predefined JSON format",
			Required:    true,
		},
	},
}
