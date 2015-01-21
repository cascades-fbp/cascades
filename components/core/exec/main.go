package main

import (
	"bytes"
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
	// flags
	cmdEndpoint    = flag.String("port.cmd", "", "Component's options port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	errorEndpoint  = flag.String("port.err", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	cmdPort, outPort, errPort *zmq.Socket
	cmdCh, outCh, errCh       chan bool
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
	cmdCh = make(chan bool)
	outCh = make(chan bool)
	errCh = make(chan bool)
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

	ports := 1
	if outPort != nil {
		ports++
	}
	if errPort != nil {
		ports++
	}

	waitCh := make(chan bool)
	cmdExitCh := make(chan bool, 1)
	go func(num int) {
		total := 0
		for {
			select {
			case v := <-cmdCh:
				if v {
					total++
				} else {
					cmdExitCh <- true
					break
				}
			case v := <-outCh:
				if !v {
					log.Println("OUT port is closed. Interrupting execution")
					exitCh <- syscall.SIGTERM
					break
				} else {
					total++
				}
			case v := <-errCh:
				if !v {
					log.Println("ERR port is closed. Interrupting execution")
					exitCh <- syscall.SIGTERM
					break
				} else {
					total++
				}
			}
			if total >= num && waitCh != nil {
				waitCh <- true
			}
		}
	}(ports)

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
		ip, err := cmdPort.RecvMessageBytes(0)
		if err != nil {
			continue
		}
		if !runtime.IsValidIP(ip) {
			continue
		}
		out, err := executeCommand(string(ip[1]))
		if err != nil {
			log.Println(err.Error())
			if errPort != nil {
				errPort.SendMessage(runtime.NewPacket([]byte(err.Error())))
			}
			continue
		}
		out = bytes.Replace(out, []byte("\n"), []byte(""), -1)
		log.Println(string(out))
		if outPort != nil {
			outPort.SendMessage(runtime.NewPacket(out))
		}

		select {
		case <-cmdExitCh:
			log.Println("CMD port is closed. Interrupting execution")
			exitCh <- syscall.SIGTERM
			break
		default:
			// IN port is still open
		}
	}
}

// validateArgs checks all required flags
func validateArgs() {
	if *cmdEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

// openPorts create ZMQ sockets and start socket monitoring loops
func openPorts() {
	cmdPort, err = utils.CreateInputPort("exec.cmd", *cmdEndpoint, cmdCh)
	utils.AssertError(err)

	if *outputEndpoint != "" {
		outPort, err = utils.CreateOutputPort("exec.out", *outputEndpoint, outCh)
		utils.AssertError(err)
	}

	if *errorEndpoint != "" {
		errPort, err = utils.CreateOutputPort("exec.err", *errorEndpoint, errCh)
		utils.AssertError(err)
	}
}

// closePorts closes all active ports and terminates ZMQ context
func closePorts() {
	log.Println("Closing ports...")
	cmdPort.Close()
	if outPort != nil {
		outPort.Close()
	}
	if errPort != nil {
		errPort.Close()
	}
	zmq.Term()
}
