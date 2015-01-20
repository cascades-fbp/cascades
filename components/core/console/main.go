package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	inCh   chan bool
	exitCh chan os.Signal
	err    error
)

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

	// Communication channels
	inCh = make(chan bool)
	exitCh = make(chan os.Signal, 1)

	// Block until all is connected
	go mainLoop()

	// Wait for the end...
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM)
	<-exitCh
	time.Sleep(1e9) // give ZMQ some time to finish
	log.Println("Stopped")
	os.Exit(0)
}

func mainLoop() {
	openPorts()
	defer closePorts()

	// Connections monitoring routine
	waitCh := make(chan bool)
	go func() {
		for {
			v := <-inCh
			if v && waitCh != nil {
				waitCh <- true
			}
			if !v {
				log.Println("IN port is closed. Interrupting execution")
				exitCh <- syscall.SIGTERM
				break
			}
		}
	}()

	log.Println("Waiting for port connections to establish... ")
	select {
	case <-waitCh:
		log.Println("Input port connected")
		waitCh = nil
	case <-time.Tick(30 * time.Second):
		log.Println("Timeout: port connections were not established within provided interval")
		exitCh <- syscall.SIGTERM
		return
	}

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

func validateArgs() {
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func openPorts() {
	inPort, err = utils.CreateInputPort("console.in", *inputEndpoint, inCh)
	utils.AssertError(err)
}

func closePorts() {
	inPort.Close()
	zmq.Term()
}
