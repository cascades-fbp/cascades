package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Sends ticks (current unix timestamps) at predefined intervals to the output channel",
	Elementary:  true,
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "INTERVAL",
			Type:        "duration",
			Description: "Configures the ticker. Accepts durations in the format: 3s, 10m, etc",
			Required:    true,
		},
	},
	Outports: []library.EntryPort{
		library.EntryPort{
			Name:        "OUT",
			Type:        "timestamp",
			Description: "Output port for sending ticks (timestamps)",
			Required:    true,
		},
	},
}
