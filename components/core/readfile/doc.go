package main

import (
	"github.com/cascades-fbp/cascades/registry"
)

var registryEntry = &registry.Entry{
	Description: `Reads a given file from the local file system line by line and emits each line into the output port. 
The output data is sent as substream with open bracket IP in the beginning and close bracket IP at the end of the stream for each file.`,
	Inports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "FILE",
			Type:        "string",
			Description: "Port for setting file path to be read from",
			Required:    true,
		},
	},
	Outports: []registry.EntryPort{
		registry.EntryPort{
			Name:        "OUT",
			Type:        "all",
			Description: "Output port for sending IPs",
			Required:    true,
		},
		registry.EntryPort{
			Name:        "ERR",
			Type:        "string",
			Description: "Error port for errors opening/reading file",
			Required:    false,
		},
	},
}
