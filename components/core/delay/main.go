package main

import (
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/cascades-fbp/cascades/runtime"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	inputEndpoint  = flag.String("port.in", "", "Component's input port endpoint")
	delayEndpoint  = flag.String("port.delay", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
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
	if *delayEndpoint == "" {
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

	//  Socket to receive messages on
	receiver, err := context.NewSocket(zmq.PULL)
	if err != nil {
		fmt.Println("Error creating socket:", err.Error())
		os.Exit(1)
	}
	defer receiver.Close()
	err = receiver.Bind(*inputEndpoint)
	if err != nil {
		fmt.Println("Error binding socket:", err.Error())
		os.Exit(1)
	}

	// Delay port socket
	delaySocket, err := context.NewSocket(zmq.PULL)
	if err != nil {
		fmt.Println("Error creating socket:", err.Error())
		os.Exit(1)
	}
	defer delaySocket.Close()
	err = delaySocket.Bind(*delayEndpoint)
	if err != nil {
		fmt.Println("Error binding socket:", err.Error())
		os.Exit(1)
	}

	//  Socket to send messages to task sink
	sender, err := context.NewSocket(zmq.PUSH)
	if err != nil {
		fmt.Println("Error creating socket:", err.Error())
		os.Exit(1)
	}
	defer sender.Close()
	err = sender.Connect(*outputEndpoint)
	if err != nil {
		fmt.Println("Error connecting to socket:", err.Error())
		os.Exit(1)
	}

	// Ctrl+C handling
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range ch {
			log.Println("Give 0MQ time to deliver before stopping...")
			time.Sleep(1e9)
			log.Println("Stopped")
			os.Exit(0)
		}
	}()

	// Monitoring setup
	err = runtime.SetupShutdownByDisconnect(context, receiver, "delay.in", ch)
	if err != nil {
		log.Println("Failed to setup monitoring:", err.Error())
		os.Exit(1)
	}

	log.Println("Started")

	var delay time.Duration
	for {
		ip, err := delaySocket.RecvMultipart(0)
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

	//  Process tasks forever
	for {
		ip, err := receiver.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			continue
		}

		time.Sleep(delay)

		sender.SendMultipart(ip, zmq.NOBLOCK)
	}
}
