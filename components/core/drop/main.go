package main

import (
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	"io/ioutil"
	"log"
	"os"
)

var (
	// Flags
	inputEndpoint = flag.String("port.in", "", "Component's input port endpoint")
	json          = flag.Bool("json", false, "Print component documentation in JSON")
	debug         = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	context *zmq.Context
	inPort  *zmq.Socket
	err     error
)

func validateArgs() {
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func openPorts() {
	context, err = zmq.NewContext()
	utils.AssertError(err)

	inPort, err = utils.CreateInputPort(context, *inputEndpoint)
	utils.AssertError(err)
}

func closePorts() {
	inPort.Close()
	context.Close()
}

func main() {
	flag.Parse()

	if *json {
		doc, _ := registryEntry.JSON()
		fmt.Println(string(doc))
		os.Exit(0)
	}

	log.SetFlags(0)
	if *debug {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	validateArgs()

	openPorts()
	defer closePorts()

	ch := utils.HandleInterruption()
	err = runtime.SetupShutdownByDisconnect(context, inPort, "drop.in", ch)
	utils.AssertError(err)

	log.Println("Started...")
	for {
		ip, err := inPort.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			log.Println("Invalid IP:", ip)
			continue
		}
	}
}
