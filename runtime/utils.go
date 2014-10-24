package runtime

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/cascades-fbp/cascades/graph"
)

func loadGraph(graphfile string) (g *graph.GraphDescription, err error) {
	data, err := ioutil.ReadFile(graphfile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read graph definition from file: %s", err.Error())
	}
	if strings.HasSuffix(graphfile, ".fbp") {
		g, err = graph.ParseFBP(data)
	} else if strings.HasSuffix(graphfile, ".json") {
		g, err = graph.ParseJSON(data)
	} else {
		return nil, fmt.Errorf("Unsupported graph format (should be .fbp or .json): %s", err.Error())
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to parse graph definition: %s", err.Error())
	}

	return g, nil
}
