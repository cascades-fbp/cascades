package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"

	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	zmq "github.com/pebbe/zmq4"
)

var (
	// Flags
	tplEndpoint    = flag.String("port.tpl", "", "Component's options port endpoint")
	inputEndpoint  = flag.String("port.in", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	tplPort, inPort, outPort *zmq.Socket
	inCh, outCh              chan bool
	exitCh                   chan os.Signal
	err                      error
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
	go func() {
		total := 0
		for {
			select {
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
			if total >= 2 && waitCh != nil {
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

	log.Println("Waiting for template...")
	var (
		t  *template.Template
		ip [][]byte
	)
	for {
		ip, err = tplPort.RecvMessageBytes(0)
		if err != nil {
			continue
		}
		if !runtime.IsValidIP(ip) {
			log.Println("Invalid IP:", ip)
			continue
		}
		t = template.New("Current template")
		t, err = t.Parse(string(ip[1]))
		if err != nil {
			log.Println("Failed to configure component:", err.Error())
			continue
		}
		break
	}
	tplPort.Close()

	log.Println("Started...")
	var (
		buf  *bytes.Buffer
		data map[string]interface{}
	)
	for {
		ip, err := inPort.RecvMessageBytes(0)
		if err != nil {
			continue
		}
		if !runtime.IsValidIP(ip) {
			continue
		}

		err = json.Unmarshal(ip[1], &data)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		buf = bytes.NewBufferString("")
		err = t.Execute(buf, data)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		outPort.SendMessage(runtime.NewPacket(buf.Bytes()))
	}
}

// validateArgs checks all required flags
func validateArgs() {
	if *tplEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
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
	tplPort, err = utils.CreateInputPort("template.tpl", *tplEndpoint, nil)
	utils.AssertError(err)

	inPort, err = utils.CreateInputPort("template.in", *inputEndpoint, inCh)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort("template.out", *outputEndpoint, outCh)
	utils.AssertError(err)
}

// closePorts closes all active ports and terminates ZMQ context
func closePorts() {
	log.Println("Closing ports...")
	tplPort.Close()
	inPort.Close()
	outPort.Close()
	zmq.Term()
}
