package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "Merges IPs from input array port into a single stream in the natural order of IPs arrival",
	Inports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "IN",
			Type:        "all",
			Description: "Input array port",
			Required:    true,
		},
	},
	Outports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OUT",
			Type:        "all",
			Description: "Output port with merged stream of IPs from input array",
			Required:    true,
		},
	},
}
