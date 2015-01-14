package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	zmq "github.com/pebbe/zmq4"
)

var (
	// Flags
	inputEndpoint  = flag.String("port.in", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port #1 endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	outPortArray []*zmq.Socket
	inPort, port *zmq.Socket
	err          error
)

func validateArgs() {
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *outputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func openPorts(termCh chan os.Signal) {
	outports := strings.Split(*outputEndpoint, ",")
	if len(outports) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	inPort, err = utils.CreateMonitoredInputPort("splitter.in", *inputEndpoint, termCh)
	utils.AssertError(err)

	outPortArray = []*zmq.Socket{}
	for i, endpoint := range outports {
		endpoint = strings.TrimSpace(endpoint)
		log.Printf("Connecting OUT[%v]=%s", i, endpoint)
		port, err = utils.CreateMonitoredOutputPort(fmt.Sprintf("splitter.out[%v]", i), endpoint, termCh)
		outPortArray = append(outPortArray, port)
	}
}

func closePorts() {
	inPort.Close()
	for _, port = range outPortArray {
		port.Close()
	}
	zmq.Term()
}

func main() {
	flag.Parse()

	if *jsonFlag {
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
			log.Println("Received invalid IP")
			continue
		}
		for _, port = range outPortArray {
			port.SendMessage(ip)
		}
	}
}
