package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	exitCh       chan os.Signal
	inCh, outCh  chan bool
	err          error
)

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

	// Communication channels
	inCh = make(chan bool)
	outCh = make(chan bool)
	exitCh = make(chan os.Signal, 1)

	// Start the communication & processing logic
	go mainLoop()

	// Wait for the end...
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM)
	<-exitCh

	closePorts()
	log.Println("Done")
}

// mainLoop initiates all ports and handles the traffic
func mainLoop() {
	openPorts()
	defer closePorts()

	waitCh := make(chan bool)
	go func(num int) {
		total := 0
		for {
			select {
			case v := <-outCh:
				if v {
					total++
				} else {
					total--
				}
			case v := <-inCh:
				if !v {
					log.Println("IN port is closed. Interrupting execution")
					exitCh <- syscall.SIGTERM
					break
				} else {
					total++
				}
			}
			if total >= num && waitCh != nil {
				waitCh <- true
			} else if total <= 1 && waitCh == nil {
				log.Println("All output ports closed. Interrupting execution")
				exitCh <- syscall.SIGTERM
				break
			}
		}
	}(len(outPortArray) + 1)

	log.Println("Waiting for port connections to establish... ")
	select {
	case <-waitCh:
		log.Println("Ports connected")
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

// validateArgs checks all required flags
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

// openPorts create ZMQ sockets and start socket monitoring loops
func openPorts() {
	outports := strings.Split(*outputEndpoint, ",")
	if len(outports) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	inPort, err = utils.CreateInputPort("splitter.in", *inputEndpoint, inCh)
	utils.AssertError(err)

	outPortArray = []*zmq.Socket{}
	for i, endpoint := range outports {
		endpoint = strings.TrimSpace(endpoint)
		log.Printf("Connecting OUT[%v]=%s", i, endpoint)
		port, err = utils.CreateOutputPort(fmt.Sprintf("splitter.out[%v]", i), endpoint, outCh)
		outPortArray = append(outPortArray, port)
	}
}

// closePorts closes all active ports and terminates ZMQ context
func closePorts() {
	inPort.Close()
	for _, port = range outPortArray {
		port.Close()
	}
	zmq.Term()
}
