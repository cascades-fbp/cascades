package main

import (
	"github.com/cascades-fbp/cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: "Simple logging component that writes everything received on the input port to standard output stream.",
	Inports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "IN",
			Type:        "all",
			Description: "Input port for logging IP",
			Required:    true,
		},
	},
}
