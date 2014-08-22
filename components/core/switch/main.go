package main

import (
	"cascades/runtime"
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	inputEndpoint  = flag.String("port.in", "", "Component's data input port endpoint")
	gateEndpoint   = flag.String("port.gate", "", "Component's triggering port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's data output port endpoint")
	json           = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")
)

func main() {
	flag.Parse()

	if *json {
		doc, _ := registryEntry.JSON()
		fmt.Println(string(doc))
		os.Exit(0)
	}

	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	flag.Parse()
	if *gateEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *outputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.SetFlags(0)
	if *debug {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	var err error
	context, _ := zmq.NewContext()
	defer context.Close()

	//  Socket to receive data on
	inSock, err := context.NewSocket(zmq.PULL)
	if err != nil {
		fmt.Println("Error creating socket:", err.Error())
		os.Exit(1)
	}
	defer inSock.Close()
	err = inSock.Bind(*inputEndpoint)
	if err != nil {
		fmt.Println("Error binding socket:", err.Error())
		os.Exit(1)
	}

	//  Socket to send messages to task sink
	outSock, err := context.NewSocket(zmq.PUSH)
	if err != nil {
		fmt.Println("Error creating socket:", err.Error())
		os.Exit(1)
	}
	defer outSock.Close()
	err = outSock.Connect(*outputEndpoint)
	if err != nil {
		fmt.Println("Error connecting to socket:", err.Error())
		os.Exit(1)
	}

	// Ctrl+C handling
	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range exitCh {
			log.Println("Give 0MQ time to deliver before stopping...")
			time.Sleep(1e9)
			log.Println("Stopped")
			os.Exit(0)
		}
	}()

	// Monitoring setup
	err = runtime.SetupShutdownByDisconnect(context, inSock, "switch.in", exitCh)
	if err != nil {
		log.Println("Failed to setup monitoring:", err.Error())
		exitCh <- syscall.SIGTERM
	}

	// Start a separate goroutine to receive gate signals and avoid stocking them
	// blocking the channel (use timeout to skip ticks if data sending is still in progress)
	tickCh := make(chan bool)
	go func() {
		//  Socket to receive signal on
		gateSock, _ := context.NewSocket(zmq.PULL)
		defer gateSock.Close()
		gateSock.Bind(*gateEndpoint)
		// Setup up monitoring
		err = runtime.SetupShutdownByDisconnect(context, gateSock, "switch.gate", exitCh)
		if err != nil {
			log.Println("[Tick routine]: Failed to setup monitoring:", err.Error())
			exitCh <- syscall.SIGTERM
		}
		for {
			log.Println("[Tick routine]: Wait for tick...")
			ip, err := gateSock.RecvMultipart(0)
			if err != nil {
				log.Println("[Tick routine]: Error receiving message:", err.Error())
				continue
			}
			if !runtime.IsValidIP(ip) {
				log.Println("[Tick routine]: Invalid IP received:", err.Error())
				continue
			}
			log.Println("[Tick routine]: Tick received")
			select {
			case tickCh <- true:
				log.Println("[Tick routine]: Main thread notified")
			case <-time.Tick(time.Duration(5) * time.Second):
				log.Println("[Tick routine]: Timeout, skipping this tick")
				continue
			}
		}
	}()

	log.Println("Started")

	var (
		ip          [][]byte
		isSubstream bool
	)
	for {
		select {
		case <-tickCh:
			log.Println("[Main routine]: Passing data through...")
			// Now read from data
			for {
				ip, err = inSock.RecvMultipart(0)
				if err != nil {
					log.Println("[Main routine]: Error receiving message:", err.Error())
					break
				}
				if !runtime.IsValidIP(ip) {
					log.Println("[Main routine]: Invalid IP received:", err.Error())
					break
				}
				outSock.SendMultipart(ip, zmq.NOBLOCK)
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
