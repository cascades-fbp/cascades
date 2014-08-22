package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "Sends out a hard-coded text, closes the out port and exits with exit code 0",
	Outports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OUT",
			Type:        "all",
			Description: "Output port for sending IPs",
			Required:    true,
		},
	},
}
