package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: "Recursively walks a given directory and sends filepaths into the output port (only files, directories are omitted)",
	Elementary:  true,
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "DIR",
			Type:        "string",
			Description: "Directory on the file system to walk",
			Required:    true,
		},
	},
	Outports: []library.EntryPort{
		library.EntryPort{
			Name:        "FILE",
			Type:        "string",
			Description: "...",
			Required:    true,
		},
		library.EntryPort{
			Name:        "ERR",
			Type:        "string",
			Description: "Error port for errors walking directory",
			Required:    false,
		},
	},
}
