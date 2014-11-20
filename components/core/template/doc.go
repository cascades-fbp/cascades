package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Fill the template string with received data from the input port and pass it to the output port",
	Elementary:  true,
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "TPL",
			Type:        "string",
			Description: "Port for configuring component with a template",
			Required:    true,
		},
		library.EntryPort{
			Name:        "IN",
			Type:        "all",
			Description: "Input port for receiving IPs",
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
