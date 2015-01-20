package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "A filter component (either file-based or memory-based) that passes through only unique IPs (keeps a complete history of IPs)",
	Elementary:  true,
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "OPTIONS",
			Type:        "json",
			Description: "Port for options to configure the component. E.g. {'storage':'file or memory', 'file':'/path/to/cache/file'}",
			Required:    true,
		},
		library.EntryPort{
			Name:        "IN",
			Type:        "string",
			Description: "Input port for receiving IPs",
			Required:    true,
		},
	},
	Outports: []library.EntryPort{
		library.EntryPort{
			Name:        "OUT",
			Type:        "string",
			Description: "Output port for sending distinct IPs",
			Required:    true,
		},
	},
}
