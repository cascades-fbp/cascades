package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Simple logging component that writes everything received on the input port to standard output stream.",
	Elementary:  true,
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "IN",
			Type:        "all",
			Description: "Input port for logging IP",
			Required:    true,
		},
	},
}
