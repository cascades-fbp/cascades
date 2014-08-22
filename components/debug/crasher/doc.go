package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "Based on pasthru component (forwarder), but exits in a couple of seconds after the start with exit code 0",
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
