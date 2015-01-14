package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"

	"github.com/cascades-fbp/cascades/library"
	"github.com/cascades-fbp/cascades/runtime"
	"github.com/codegangsta/cli"
	zmq "github.com/pebbe/zmq4"
)

func run(c *cli.Context) {
	if len(c.Args()) != 1 {
		fmt.Printf("Incorrect Usage. You need to provide a path to a graph as argument!\n\n")
		cli.ShowAppHelp(c)
		return
	}

	// read components library file if exists
	data, err := ioutil.ReadFile(c.GlobalString("file"))
	if err != nil {
		fmt.Printf("Failed to read catalogue file: %s\n", err.Error())
		return
	}
	var db library.JSONLibrary
	err = json.Unmarshal(data, &db)
	if err != nil {
		fmt.Printf("Failed to parse catalogue file: %s\n", err.Error())
		return
	}

	// create runtime for a graph, validate and execute it
	scheduler := runtime.NewRuntime(db, uint(c.Int("port")))
	scheduler.Debug = c.GlobalBool("debug")
	err = scheduler.LoadGraph(c.Args().First())
	if err != nil {
		fmt.Printf("Failed to load/flatten graph: %s\n", err.Error())
		return
	}

	if scheduler.Debug {
		scheduler.PrintGraph()
	}

	if c.Bool("dry") {
		return
	}

	// Start the network
	go scheduler.Start()

	// Shutdown ZMQ upon shutdown
	defer zmq.Term()

	// Ctrl+C handling
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	for {
		select {
		case <-ch:
			go scheduler.Shutdown()
		case <-scheduler.Done:
			fmt.Println("Stopped")
			os.Exit(0)
		}
	}
}
