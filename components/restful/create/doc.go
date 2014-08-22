package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "RESTful create entity handler",
	Inports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "TYPE",
			Type:        "string",
			Description: "Content-type of the data input (application/json, application/xml, etc)",
			Required:    true,
		},
		registry.EntryPort{
			Name:        "DSN",
			Type:        "string",
			Description: "DSN for connecting to persistance storage (i.e. mongodb://127.0.0.1:27017/collection)",
			Required:    true,
		},
		registry.EntryPort{
			Name:        "REQUIRED",
			Type:        "string",
			Description: "Comma-separated list of required keys",
			Required:    false,
		},
		registry.EntryPort{
			Name:        "UNIQUE",
			Type:        "string",
			Description: "Comma-separated list of unique keys",
			Required:    false,
		},
		registry.EntryPort{
			Name:        "IN",
			Type:        "substream",
			Description: "Input port for receiving request substreams",
			Required:    true,
		},
	},
	Outports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OUT",
			Type:        "substream",
			Description: "Output port for emitting response substreams",
			Required:    true,
		},
		registry.EntryPort{
			Name:        "ERR",
			Type:        "substream",
			Description: "Output port for emitting response substreams",
			Required:    true,
		},
	},
}
