package graph

import (
	"fmt"
)

// Description describes FBP network
type Description struct {
	Properties  map[string]string  `json:"properties"`
	Processes   map[string]Process `json:"processes`
	Connections []Connection       `json:"connections"`
	Inports     []Export           `json:"inports"`
	Outports    []Export           `json:"outports`
}

// Process of the network
type Process struct {
	Component string
	Metadata  map[string]string `json:"omitempty"`
}

// Connection between processes in the network
type Connection struct {
	Data     string            `json:"data,omitempty"`
	Src      *Endpoint         `json:"src,omitempty"`
	Tgt      *Endpoint         `json:"tgt,omitempty"`
	Metadata map[string]string `json:"omitempty"`
}

// Endpoint of the process
type Endpoint struct {
	Process string `json:"process"`
	Port    string `json:"port"`
	Index   *int   `json:"index,omitempty"`
}

// Export exported port of the process
type Export struct {
	Private string `json:"private"`
	Public  string `json:"public"`
}

// NewDescription is a Description constructor
func NewDescription() *Description {
	return &Description{
		Properties:  make(map[string]string),
		Processes:   make(map[string]Process),
		Connections: []Connection{},
		Inports:     []Export{},
		Outports:    []Export{},
	}
}

func (endpoint *Endpoint) String(isInput bool) string {
	if isInput {
		if endpoint.Index != nil {
			return fmt.Sprintf("%s %s[%v]", endpoint.Process, endpoint.Port, *endpoint.Index)
		}
		return fmt.Sprintf("%s %s", endpoint.Process, endpoint.Port)
	}
	if endpoint.Index != nil {
		return fmt.Sprintf("%s[%v] %s", endpoint.Port, *endpoint.Index, endpoint.Process)
	}
	return fmt.Sprintf("%s %s", endpoint.Port, endpoint.Process)
}

func (process *Process) String() string {
	return "(" + process.Component + ")"
}

func (connection *Connection) String() string {
	result := ""
	if connection.Src == nil {
		result = "'" + connection.Data + "'" + " -> " + connection.Tgt.String(false)
	} else {
		result = connection.Src.String(true) + " -> " + connection.Tgt.String(false)
	}
	return result
}
