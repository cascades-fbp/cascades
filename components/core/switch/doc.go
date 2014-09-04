package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Receives IP on the IN port and passes it to OUT only when GATE receives an IP",
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "IN",
			Type:        "all",
			Description: "Data port",
			Required:    true,
		},
		library.EntryPort{
			Name:        "GATE",
			Type:        "all",
			Description: "Gate port to pass IP from data port to the output",
			Required:    true,
		},
	},
	Outports: []library.EntryPort{
		library.EntryPort{
			Name:        "OUT",
			Type:        "all",
			Description: "Output port",
			Required:    true,
		},
	},
}
