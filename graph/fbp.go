package graph

import (
	"github.com/oleksandr/fbp"
)

// Parses a given definition in NoFlo's .fbp DSL and returns
// unified GraphDescription structure
func ParseFBP(definition []byte) (*GraphDescription, error) {
	// Parse .fbp file
	parser := &fbp.Fbp{Buffer: string(definition)}
	parser.Init()
	err := parser.Parse()
	if err != nil {
		return nil, err
	}
	parser.Execute()
	if err = parser.Validate(); err != nil {
		return nil, err
	}

	// Populate standard graph description structure
	graph := NewGraphDescription()
	for _, p := range parser.Processes {
		graph.Processes[p.Name] = Process{
			Component: p.Component,
			Metadata:  p.Metadata,
		}
	}
	for _, c := range parser.Connections {
		connection := Connection{}
		if c.Source != nil {
			connection.Data = ""
			connection.Src = &Endpoint{
				Process: c.Source.Process,
				Port:    c.Source.Port,
				Index:   c.Source.Index,
			}
		} else {
			connection.Data = c.Data
		}
		connection.Tgt = &Endpoint{
			Process: c.Target.Process,
			Port:    c.Target.Port,
			Index:   c.Target.Index,
		}
		connection.Metadata = nil
		graph.Connections = append(graph.Connections, connection)
	}
	for pub, endpoint := range parser.Inports {
		export := Export{
			Private: endpoint.Process + "." + endpoint.Port,
			Public:  pub,
		}
		graph.Inports = append(graph.Inports, export)
	}
	for pub, endpoint := range parser.Outports {
		export := Export{
			Private: endpoint.Process + "." + endpoint.Port,
			Public:  pub,
		}
		graph.Outports = append(graph.Outports, export)
	}

	return graph, nil
}
