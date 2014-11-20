package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Uses a given RegEx with named capturing groups for analyzing an input string. Outputs the matched map in JSON",
	Elementary:  true,
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "PATTERN",
			Type:        "string",
			Description: "Port for RegExp patte with named capturing groups",
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
			Name:        "MAP",
			Type:        "json",
			Description: "Output port for captured submatching map in JSON",
			Required:    true,
		},
	},
}
