package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"

	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	zmq "github.com/pebbe/zmq4"
)

var (
	// Flags
	inputEndpoint  = flag.String("port.in", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	inPort, outPort *zmq.Socket
	inCh, outCh     chan bool
	err             error
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
	inPort, err = utils.CreateInputPort("passthru.in", *inputEndpoint, inCh)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort("passthru.out", *outputEndpoint, outCh)
	utils.AssertError(err)
}

func closePorts() {
	inPort.Close()
	outPort.Close()
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
	inCh = make(chan bool)
	outCh = make(chan bool)
	go func() {
		select {
		case <-inCh:
			log.Println("IN port is closed. Interrupting execution")
			ch <- syscall.SIGTERM
		case <-outCh:
			log.Println("OUT port is closed. Interrupting execution")
			ch <- syscall.SIGTERM
		}
	}()

	openPorts()
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
		outPort.SendMessage(ip)
	}
}
