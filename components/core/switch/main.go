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
	inputEndpoint  = flag.String("port.in", "", "Component's data input port endpoint")
	gateEndpoint   = flag.String("port.gate", "", "Component's triggering port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's data output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	inPort, gatePort, outPort *zmq.Socket
	inCh, gateCh, outCh       chan bool
	exitCh                    chan os.Signal
	err                       error
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
	gateCh = make(chan bool)
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
	// Start a separate goroutine to receive gate signals and avoid stocking them
	// blocking the channel (use timeout to skip ticks if data sending is still in progress)
	tickCh := make(chan bool)
	go func() {
		//  Socket to receive signal on
		gatePort, err = utils.CreateInputPort("switch.gate", *gateEndpoint, gateCh)
		utils.AssertError(err)
		defer gatePort.Close()

		for {
			log.Println("[Gate routine]: Wait for IP on gate port...")
			ip, err := gatePort.RecvMessageBytes(0)
			if err != nil {
				continue
			}
			if !runtime.IsValidIP(ip) {
				log.Println("[Gate routine]: Invalid IP received:", err.Error())
				continue
			}
			select {
			case tickCh <- true:
				log.Println("[Tick routine]: Main thread notified")
			case <-time.Tick(time.Duration(5) * time.Second):
				log.Println("[Tick routine]: Timeout, skipping this tick")
				continue
			}
		}
	}()

	openPorts()
	defer closePorts()

	waitCh := make(chan bool)
	go func() {
		total := 0
		for {
			select {
			case v := <-gateCh:
				if !v {
					log.Println("GATE port is closed. Interrupting execution")
					exitCh <- syscall.SIGTERM
					break
				} else {
					total++
				}
			case v := <-inCh:
				if !v {
					log.Println("IN port is closed. Interrupting execution")
					exitCh <- syscall.SIGTERM
					break
				} else {
					total++
				}
			case v := <-outCh:
				if !v {
					log.Println("OUT port is closed. Interrupting execution")
					exitCh <- syscall.SIGTERM
					break
				} else {
					total++
				}
			}
			if total >= 3 && waitCh != nil {
				waitCh <- true
			}
		}
	}()

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
	var (
		ip          [][]byte
		isSubstream bool
	)
	for {
		select {
		case <-tickCh:
			log.Println("[Main routine]: Passing data through...")
			// Now read from in port (if it's a substream pass it as the whole)
			for {
				ip, err = inPort.RecvMessageBytes(0)
				if err != nil {
					break
				}
				if !runtime.IsValidIP(ip) {
					log.Println("[Main routine]: Invalid IP received:", err.Error())
					break
				}
				outPort.SendMessage(ip)
				if runtime.IsOpenBracket(ip) {
					isSubstream = true
					log.Println("[Main routine]: Substream BEGIN")
				}
				if runtime.IsPacket(ip) && !isSubstream {
					log.Println("[Main routine]: Received data as NOT part of substream")
					break
				}
				if runtime.IsCloseBracket(ip) {
					isSubstream = false
					log.Println("[Main routine]: Substream END")
					break
				}
				log.Println("[Main routine]: Done")
			}
		}
	}
}

// validateArgs checks all required flags
func validateArgs() {
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *gateEndpoint == "" {
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
	inPort, err = utils.CreateInputPort("switch.in", *inputEndpoint, inCh)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort("switch.out", *outputEndpoint, outCh)
	utils.AssertError(err)
}

// closePorts closes all active ports and terminates ZMQ context
func closePorts() {
	inPort.Close()
	outPort.Close()
	zmq.Term()
}
