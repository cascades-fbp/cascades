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
	inputEndpoint  = flag.String("port.in", "", "Component's input port #1 endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	inPortArray   []*zmq.Socket
	outPort, port *zmq.Socket
	poller        *zmq.Poller
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

	inPortArray = []*zmq.Socket{}
	poller = zmq.NewPoller()

	for i, endpoint := range inports {
		endpoint = strings.TrimSpace(endpoint)
		log.Printf("Binding OUT[%v]=%s", i, endpoint)
		port, err = utils.CreateInputPort(endpoint)
		utils.AssertError(err)
		inPortArray = append(inPortArray, port)
		poller.Add(port, zmq.POLLIN)
	}

	outPort, err = utils.CreateOutputPort(*outputEndpoint)
	utils.AssertError(err)
}

func closePorts() {
	for _, port = range inPortArray {
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

	openPorts()
	defer closePorts()

	utils.HandleInterruption()

	log.Println("Started...")
	var (
		ip [][]byte
	)

	for {
		results, err := poller.Poll(-1)
		if err != nil {
			log.Println("Error polling ports:", err.Error())
			continue
		}

		for _, r := range results {
			if r.Socket == nil {
				log.Println("ERROR: could not find socket in the polling results")
				continue
			}
			ip, err = r.Socket.RecvMessageBytes(0)
			if err != nil {
				log.Println("Error receiving message:", err.Error())
				continue
			}
			if !runtime.IsValidIP(ip) {
				log.Println("Received invalid IP")
				continue
			}

			outPort.SendMessage(ip)
		}
	}
}
