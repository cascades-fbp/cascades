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
	inputEndpoint  = flag.String("port.in", "", "Component's input port #1 endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	inPortArray   []*zmq.Socket
	outPort, port *zmq.Socket
	inCh, outCh   chan bool
	exitCh        chan os.Signal
	poller        *zmq.Poller
	err           error
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
			case v := <-inCh:
				if v {
					total++
				} else {
					total--
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
			if total >= num && waitCh != nil {
				waitCh <- true
			} else if total <= 1 && waitCh == nil {
				log.Println("All IN ports closed. Interrupting execution")
				exitCh <- syscall.SIGTERM
				break
			}
		}
	}(len(inPortArray) + 1)

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
	var ip [][]byte

	for {
		results, err := poller.Poll(-1)
		if err != nil {
			continue
		}

		for _, r := range results {
			if r.Socket == nil {
				log.Println("ERROR: could not find socket in the polling results")
				continue
			}
			ip, err = r.Socket.RecvMessageBytes(0)
			if err != nil {
				continue
			}
			if !runtime.IsValidIP(ip) {
				log.Println("Received invalid IP")
				continue
			}

			if *debug {
				for i, s := range inPortArray {
					if s == r.Socket {
						log.Printf("Data from port IN[%v]: %#v", i, string(ip[1]))
					}
				}
			}

			outPort.SendMessage(ip)
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
	inports := strings.Split(*inputEndpoint, ",")
	if len(inports) == 0 {
		flag.Usage()
		exitCh <- syscall.SIGTERM
		return
	}

	inPortArray = []*zmq.Socket{}
	poller = zmq.NewPoller()

	for i, endpoint := range inports {
		endpoint = strings.TrimSpace(endpoint)
		log.Printf("Binding IN[%v]=%s", i, endpoint)
		port, err = utils.CreateInputPort(fmt.Sprintf("joiner.in[%v]", i), endpoint, inCh)
		utils.AssertError(err)
		inPortArray = append(inPortArray, port)
		poller.Add(port, zmq.POLLIN)
	}

	outPort, err = utils.CreateOutputPort("joiner.out", *outputEndpoint, outCh)
	utils.AssertError(err)
}

// closePorts closes all active ports and terminates ZMQ context
func closePorts() {
	log.Println("Closing ports...")
	for _, port = range inPortArray {
		port.Close()
	}
	outPort.Close()
	zmq.Term()
}
