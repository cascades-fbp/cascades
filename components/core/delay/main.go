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
	inputEndpoint  = flag.String("port.in", "", "Component's input port endpoint")
	delayEndpoint  = flag.String("port.delay", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	json           = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	context                    *zmq.Context
	inPort, delayPort, outPort *zmq.Socket
	err                        error
)

func validateArgs() {
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *delayEndpoint == "" {
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

	inPort, err = utils.CreateInputPort(context, *inputEndpoint)
	utils.AssertError(err)

	delayPort, err = utils.CreateInputPort(context, *delayEndpoint)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort(context, *outputEndpoint)
	utils.AssertError(err)
}

func closePorts() {
	inPort.Close()
	delayPort.Close()
	outPort.Close()
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
	err = runtime.SetupShutdownByDisconnect(context, inPort, "delay.in", ch)
	utils.AssertError(err)

	log.Println("Waiting for configuration IP...")
	var delay time.Duration
	for {
		ip, err := delayPort.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving IP:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) || !runtime.IsPacket(ip) {
			continue
		}
		delay, err = time.ParseDuration(string(ip[1]))
		if err != nil {
			log.Println("Error parsing duration from IP:", err.Error())
			continue
		}
		break
	}

	log.Println("Started...")
	for {
		ip, err := inPort.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			continue
		}
		time.Sleep(delay)
		outPort.SendMultipart(ip, 0)
	}
}
