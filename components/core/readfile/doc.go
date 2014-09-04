package main

import (
	"github.com/cascades-fbp/cascades/library"
)

var registryEntry = &library.Entry{
	Description: `Reads a given file from the local file system line by line and emits each line into the output port. 
The output data is sent as substream with open bracket IP in the beginning and close bracket IP at the end of the stream for each file.`,
	Inports: []library.EntryPort{
		library.EntryPort{
			Name:        "FILE",
			Type:        "string",
			Description: "Port for setting file path to be read from",
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
		library.EntryPort{
			Name:        "ERR",
			Type:        "string",
			Description: "Error port for errors opening/reading file",
			Required:    false,
		},
	},
}
