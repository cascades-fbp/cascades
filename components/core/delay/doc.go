package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Forwards received IP to the output with a specified delay",
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "IN",
			Type:        "all",
			Description: "Input port for receiving IPs",
			Required:    true,
		},
		library.EntryPort{
			Name:        "INTERVAL",
			Type:        "duration",
			Description: "Configures the delay. Accepts durations in the format: 3s, 10m, etc",
			Required:    true,
		}},
	Outports: []library.EntryPort{
		library.EntryPort{
			Name:        "OUT",
			Type:        "all",
			Description: "Output port for sending IPs",
			Required:    true,
		},
	},
}
