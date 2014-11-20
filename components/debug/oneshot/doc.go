package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Sends out a hard-coded text, closes the out port and exits with exit code 0",
	Elementary:  true,
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "VALUE",
			Type:        "all",
			Description: "Configures the one-shot component to send exactly the same IP, close ports and exit.",
			Required:    true,
		},
	},
	Outports: []library.EntryPort{
		library.EntryPort{
			Name:        "OUT",
			Type:        "all",
			Description: "Output port for sending IPs",
			Required:    true,
		},
	},
}
