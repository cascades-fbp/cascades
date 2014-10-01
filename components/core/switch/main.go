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
	inputEndpoint  = flag.String("port.in", "", "Component's data input port endpoint")
	gateEndpoint   = flag.String("port.gate", "", "Component's triggering port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's data output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	context                   *zmq.Context
	inPort, gatePort, outPort *zmq.Socket
	err                       error
)

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

func openPorts() {
	context, err = zmq.NewContext()
	utils.AssertError(err)

	inPort, err = utils.CreateInputPort(context, *inputEndpoint)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort(context, *outputEndpoint)
	utils.AssertError(err)
}

func closePorts() {
	inPort.Close()
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

	ch := utils.HandleInterruption()
	err = runtime.SetupShutdownByDisconnect(context, inPort, "switch.in", ch)
	utils.AssertError(err)

	// Start a separate goroutine to receive gate signals and avoid stocking them
	// blocking the channel (use timeout to skip ticks if data sending is still in progress)
	tickCh := make(chan bool)
	go func() {
		//  Socket to receive signal on
		gatePort, err = utils.CreateInputPort(context, *gateEndpoint)
		utils.AssertError(err)
		defer gatePort.Close()

		// Setup up monitoring
		err = runtime.SetupShutdownByDisconnect(context, inPort, "switch.gate", ch)
		utils.AssertError(err)

		for {
			log.Println("[Gate routine]: Wait for IP on gate port...")
			ip, err := gatePort.RecvMultipart(0)
			if err != nil {
				log.Println("[Gate routine]: Error receiving message:", err.Error())
				continue
			}
			if !runtime.IsValidIP(ip) {
				log.Println("[Gate routine]: Invalid IP received:", err.Error())
				continue
			}
			log.Println("[Gate routine]: IP received")
			select {
			case tickCh <- true:
				log.Println("[Tick routine]: Main thread notified")
			case <-time.Tick(time.Duration(5) * time.Second):
				log.Println("[Tick routine]: Timeout, skipping this tick")
				continue
			}
		}
	}()

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
				ip, err = inPort.RecvMultipart(0)
				if err != nil {
					log.Println("[Main routine]: Error receiving message:", err.Error())
					break
				}
				if !runtime.IsValidIP(ip) {
					log.Println("[Main routine]: Invalid IP received:", err.Error())
					break
				}
				outPort.SendMultipart(ip, 0)
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
