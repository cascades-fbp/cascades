package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Sends out a hard-coded text, closes the out port and exits with exit code 0",
	Outports: []library.EntryPort{
		library.EntryPort{
			Name:        "OUT",
			Type:        "all",
			Description: "Output port for sending IPs",
			Required:    true,
		},
	},
}
