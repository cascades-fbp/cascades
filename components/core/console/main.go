package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	zmq "github.com/pebbe/zmq4"
)

var (
	// Flags
	inputEndpoint = flag.String("port.in", "", "Component's input port endpoint")
	json          = flag.Bool("json", false, "Print component documentation in JSON")
	debug         = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	inPort *zmq.Socket
	err    error
)

func validateArgs() {
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func openPorts(termCh chan os.Signal) {
	inPort, err = utils.CreateMonitoredInputPort("console.in", *inputEndpoint, termCh)
	utils.AssertError(err)
}

func closePorts() {
	inPort.Close()
	zmq.Term()
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

	ch := utils.HandleInterruption()
	openPorts(ch)
	defer closePorts()

	log.Println("Started...")
	for {
		ip, err := inPort.RecvMessageBytes(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			continue
		}
		switch {
		case runtime.IsPacket(ip):
			fmt.Println("IP:", string(ip[1]))
		case runtime.IsOpenBracket(ip):
			fmt.Println("[")
		case runtime.IsCloseBracket(ip):
			fmt.Println("]")
		}
	}
}
