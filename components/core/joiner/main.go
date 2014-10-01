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
	"strings"
)

var (
	// Flags
	inputEndpoint  = flag.String("port.in", "", "Component's input port #1 endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	context       *zmq.Context
	inPortArray   []*zmq.Socket
	outPort, port *zmq.Socket
	pollItems     zmq.PollItems
	err           error
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

func openPorts() {
	inports := strings.Split(*inputEndpoint, ",")
	if len(inports) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	context, err = zmq.NewContext()
	utils.AssertError(err)

	inPortArray = []*zmq.Socket{}
	pollItems = zmq.PollItems{}
	for i, endpoint := range inports {
		endpoint = strings.TrimSpace(endpoint)
		log.Printf("Binding OUT[%v]=%s", i, endpoint)
		port, err = utils.CreateInputPort(context, endpoint)
		utils.AssertError(err)
		inPortArray = append(inPortArray, port)
		pollItems = append(pollItems, zmq.PollItem{Socket: port, Events: zmq.POLLIN})
	}

	outPort, err = utils.CreateOutputPort(context, *outputEndpoint)
	utils.AssertError(err)
}

func closePorts() {
	for _, port = range inPortArray {
		port.Close()
	}
	context.Close()
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

	openPorts()
	defer closePorts()

	utils.HandleInterruption()

	log.Println("Started...")
	var (
		ip [][]byte
	)
	for {
		_, err = zmq.Poll(pollItems, -1)
		if err != nil {
			log.Println("Error polling ports:", err.Error())
			continue
		}

		port = nil
		for _, item := range pollItems {
			if item.REvents&zmq.POLLIN != 0 {
				port = item.Socket
				break
			}
		}
		if port == nil {
			log.Println("ERROR: could not find port in polling items array")
			continue
		}

		ip, err = port.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			log.Println("Received invalid IP")
			continue
		}

		outPort.SendMultipart(ip, 0)
	}
}
