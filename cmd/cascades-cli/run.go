package main

import (
	"encoding/json"
	"github.com/cascades-fbp/cascades/log"
	"github.com/cascades-fbp/cascades/registry"
	"github.com/cascades-fbp/cascades/runtime"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/signal"
)

//
// Run command
//
var cmdRun = &cobra.Command{
	Use:   "run [graph]",
	Short: "Runs a given graph defined in the .fbp or .json formats",
	Run:   cmdRunFunc,
}

//
// Run command handler function
//
func cmdRunFunc(cmd *cobra.Command, args []string) {
	// check the graph argument
	if len(args) != 1 {
		cmd.Usage()
		return
	}

	// parse index if exists
	data, err := ioutil.ReadFile(indexFilepath)
	if err != nil {
		log.ErrorOutput("Failed to read existing index file:" + err.Error())
		return
	}
	var db registry.JSONRegistry
	err = json.Unmarshal(data, &db)
	if err != nil {
		log.ErrorOutput("Failed to parse index file:" + err.Error())
		return
	}

	// Create runtime for a graph, validate and execute it
	runtime := runtime.NewRuntime(db, tcpPort)
	runtime.Debug = debug
	err = runtime.LoadGraph(args[0])
	if err != nil {
		log.ErrorOutput("Failed to load/flatten graph:" + err.Error())
		return
	}

	// Debug: show the graph
	if debug {
		runtime.PrintGraph()
	}

	if dryRun {
		return
	}

	// Start the network
	go runtime.Start()

	// Ctrl+C handling
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	for {
		select {
		case <-ch:
			go runtime.Shutdown()
		case <-runtime.Done:
			log.SystemOutput("Stopped")
			os.Exit(0)
		}
	}
}
