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
	"time"
)

var (
	// Flags
	intervalEndpoint = flag.String("port.interval", "", "Component's input port endpoint")
	outputEndpoint   = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag         = flag.Bool("json", false, "Print component documentation in JSON")
	debug            = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	context               *zmq.Context
	intervalPort, outPort *zmq.Socket
	err                   error
)

func validateArgs() {
	if *intervalEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *outputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func openPorts() {
	context, err = zmq.NewContext()
	utils.AssertError(err)

	intervalPort, err = utils.CreateInputPort(context, *intervalEndpoint)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort(context, *outputEndpoint)
	utils.AssertError(err)
}

func closePorts() {
	intervalPort.Close()
	outPort.Close()
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

	log.Println("Wait for configuration IP...")
	var interval time.Duration
	for {
		ip, err := intervalPort.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving IP:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) || !runtime.IsPacket(ip) {
			continue
		}
		interval, err = time.ParseDuration(string(ip[1]))
		if err != nil {
			log.Println("Error parsing duration from IP:", err.Error())
			continue
		}
		break
	}

	utils.HandleInterruption()

	log.Println("Started...")
	ticker := time.NewTicker(interval)
	log.Printf("Configured to tick with interval: %v", interval)
	for v := range ticker.C {
		msg := fmt.Sprintf("%v", v.Unix())
		log.Println(msg)
		outPort.SendMultipart(runtime.NewPacket([]byte(msg)), 0)
	}
}
