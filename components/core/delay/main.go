package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	zmq "github.com/pebbe/zmq4"
)

var (
	// Flags
	inputEndpoint  = flag.String("port.in", "", "Component's input port endpoint")
	delayEndpoint  = flag.String("port.delay", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	json           = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	inPort, delayPort, outPort *zmq.Socket
	inCh, outCh                chan bool
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
	inPort, err = utils.CreateInputPort("delay.in", *inputEndpoint, inCh)
	utils.AssertError(err)

	delayPort, err = utils.CreateInputPort("delay.delay", *delayEndpoint, nil)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort("delay.out", *outputEndpoint, outCh)
	utils.AssertError(err)
}

func closePorts() {
	inPort.Close()
	delayPort.Close()
	outPort.Close()
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

	log.Println("Waiting for configuration IP...")
	var delay time.Duration
	for {
		ip, err := delayPort.RecvMessageBytes(0)
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
	delayPort.Close()

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
		time.Sleep(delay)
		outPort.SendMessage(ip)
	}
}
