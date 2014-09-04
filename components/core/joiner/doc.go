package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Merges IPs from input array port into a single stream in the natural order of IPs arrival",
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "IN",
			Type:        "all",
			Description: "Input array port",
			Required:    true,
		},
	},
	Outports: []library.EntryPort{
		library.EntryPort{
			Name:        "OUT",
			Type:        "all",
			Description: "Output port with merged stream of IPs from input array",
			Required:    true,
		},
	},
}
