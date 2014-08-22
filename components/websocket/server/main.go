package main

import (
	"cascades/components/websocket/utils"
	"cascades/runtime"
	"code.google.com/p/go.net/websocket"
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	optionsEndpoint = flag.String("port.options", "", "Component's options port endpoint")
	inputEndpoint   = flag.String("port.in", "", "Component's input port endpoint")
	outputEndpoint  = flag.String("port.out", "", "Component's output port endpoint")
	json            = flag.Bool("json", false, "Print component documentation in JSON")
	debug           = flag.Bool("debug", false, "Enable debug mode")
)

func main() {
	flag.Parse()

	if *json {
		doc, _ := registryEntry.JSON()
		fmt.Println(string(doc))
		os.Exit(0)
	}

	if *optionsEndpoint == "" {
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

	log.SetFlags(0)
	if *debug {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	context, _ := zmq.NewContext()
	defer context.Close()

	// Options
	options, _ := context.NewSocket(zmq.PULL)
	defer options.Close()
	options.Bind(*optionsEndpoint)

	// Ctrl+C handling
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range ch {
			log.Println("Stopped")
			os.Exit(0)
		}
	}()

	// Wait for the configuration on the options port
	var bindAddr string
	for {
		ip, err := options.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving IP:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) || !runtime.IsPacket(ip) {
			continue
		}
		bindAddr = string(ip[1])
		break
	}
	options.Close()

	// Receiver routine
	go func() {
		//  Socket to receive messages on
		receiver, _ := context.NewSocket(zmq.PULL)
		defer receiver.Close()
		receiver.Bind(*inputEndpoint)
		// Process incoming message forever
		for {
			ip, err := receiver.RecvMultipart(0)
			if err != nil {
				log.Println("Error receiving message:", err.Error())
				continue
			}
			if !runtime.IsValidIP(ip) {
				continue
			}
			msg, err := utils.IP2Message(ip)
			if err != nil {
				log.Println("Failed to convert IP to Message:", err.Error())
				continue
			}
			log.Printf("Received response: %#v\n", msg)
			DefaultHub.Outgoing <- *msg
		}
	}()

	// Sender routine
	go func() {
		//  Socket to send messages to
		sender, _ := context.NewSocket(zmq.PUSH)
		defer sender.Close()
		sender.Connect(*outputEndpoint)
		for msg := range DefaultHub.Incoming {
			log.Printf("Received data from connection: %#v\n", msg)
			ip, err := utils.Message2IP(&msg)
			if err != nil {
				log.Println("Failed to convert Message to IP:", err.Error())
				continue
			}
			sender.SendMultipart(ip, zmq.NOBLOCK)
		}
	}()

	// Configure & start websocket server
	http.Handle("/", websocket.Handler(WebHandler))
	go DefaultHub.Start()

	// Listen & serve
	log.Printf("Listening %v", bindAddr)
	log.Printf("Web-socket endpoint: ws://%s/", bindAddr)
	if err := http.ListenAndServe(bindAddr, nil); err != nil {
		log.Fatal("ListenAndServe Error:", err)
	}
}
