package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "Sends ticks (current unix timestamps) at predefined intervals to the output channel",
	Inports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "INTERVAL",
			Type:        "duration",
			Description: "Configures the ticker. Accepts durations in the format: 3s, 10m, etc",
			Required:    true,
		},
	},
	Outports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OUT",
			Type:        "timestamp",
			Description: "Output port for sending ticks (timestamps)",
			Required:    true,
		},
	},
}
