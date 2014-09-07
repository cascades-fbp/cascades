package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Execute a given command",
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "CMD",
			Type:        "string",
			Description: "Port for configuring a command to execute",
			Required:    true,
		},
	},
	Outports: []library.EntryPort{
		library.EntryPort{
			Name:        "OUT",
			Type:        "string",
			Description: "Output port for sending IPs",
			Required:    false,
		},
		library.EntryPort{
			Name:        "ERR",
			Type:        "string",
			Description: "Output port for errors",
			Required:    false,
		},
	},
}
