package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Recursively monitors for changes in the given directory",
	Elementary:  true,
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "DIR",
			Type:        "string",
			Description: "Directory on the file system to watch",
			Required:    true,
		},
	},
	Outports: []library.EntryPort{
		library.EntryPort{
			Name:        "CREATED",
			Type:        "string",
			Description: "Created file path",
			Required:    true,
		},
		library.EntryPort{
			Name:        "ERR",
			Type:        "string",
			Description: "Error port for errors watching directory",
			Required:    false,
		},
	},
}
