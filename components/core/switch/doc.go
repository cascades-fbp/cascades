package main

import (
	"cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "Receives IP on the IN port and passes it to OUT only when GATE receives an IP",
	Inports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "IN",
			Type:        "all",
			Description: "Data port",
			Required:    true,
		},
		registry.EntryPort{
			Name:        "GATE",
			Type:        "all",
			Description: "Gate port to pass IP from data port to the output",
			Required:    true,
		},
	},
	Outports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OUT",
			Type:        "all",
			Description: "Output port",
			Required:    true,
		},
	},
}
