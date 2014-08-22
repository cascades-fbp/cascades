package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "Copies received IP and sends a copy to each connected out port",
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
			Description: "Output array port",
			Required:    true,
			Addressable: true,
		},
	},
}
