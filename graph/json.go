package graph

import (
	"encoding/json"
)

//
// Parses a given definition in NoFlo's .JSON and returns
// unified GraphDescription structure
//
func ParseJSON(definition []byte) (*GraphDescription, error) {
	var graph GraphDescription
	err := json.Unmarshal(definition, &graph)
	if err != nil {
		return nil, err
	}

	return &graph, nil
}
