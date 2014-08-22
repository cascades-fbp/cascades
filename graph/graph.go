package graph

import (
	"fmt"
)

type GraphDescription struct {
	Properties  map[string]string  `json:"properties"`
	Processes   map[string]Process `json:"processes`
	Connections []Connection       `json:"connections"`
	Inports     []Export           `json:"inports"`
	Outports    []Export           `json:"outports`
}

type Process struct {
	Component string
	Metadata  map[string]string `json:"omitempty"`
}

type Connection struct {
	Data     string            `json:"data,omitempty"`
	Src      *Endpoint         `json:"src,omitempty"`
	Tgt      *Endpoint         `json:"tgt,omitempty"`
	Metadata map[string]string `json:"omitempty"`
}

type Endpoint struct {
	Process string `json:"process"`
	Port    string `json:"port"`
	Index   *int   `json:"index,omitempty"`
}

type Export struct {
	Private string `json:"private"`
	Public  string `json:"public"`
}

func NewGraphDescription() *GraphDescription {
	return &GraphDescription{
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
