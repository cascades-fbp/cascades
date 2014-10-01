package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Packets dropper. Simply consumes IPs from the input port and 'deletes' them.",
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "IN",
			Type:        "all",
			Description: "Input port for IP",
			Required:    true,
		},
	},
}
