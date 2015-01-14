package graph

import (
	"encoding/json"
)

// ParseJSON parses a given definition in NoFlo's .JSON and returns
// unified Description structure
func ParseJSON(definition []byte) (*Description, error) {
	var graph Description
	err := json.Unmarshal(definition, &graph)
	if err != nil {
		return nil, err
	}

	return &graph, nil
}
