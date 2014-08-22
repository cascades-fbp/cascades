package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "Forwards received IP to the output without any modifications",
	Inports: []registry.EntryPort{
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
