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
	"strings"
	"syscall"
	"time"
)

var (
	inputEndpoint  = flag.String("port.in", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port #1 endpoint")
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

	outports := strings.Split(*outputEndpoint, ",")
	if len(outports) == 0 {
		flag.Usage()
		os.Exit(1)
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

	//  Socket to send messages
	var (
		socket *zmq.Socket
	)
	sendSockets := []*zmq.Socket{}
	for i, endpoint := range outports {
		socket, err = context.NewSocket(zmq.PUSH)
		if err != nil {
			fmt.Println("Error creating socket:", err.Error())
			os.Exit(1)
		}
		//defer socket.Close()
		endpoint = strings.TrimSpace(endpoint)
		log.Printf("Connecting OUT[%v]=%s", i, endpoint)
		err = socket.Connect(endpoint)
		if err != nil {
			fmt.Println("Error connecting to socket:", err.Error())
			os.Exit(1)
		}
		sendSockets = append(sendSockets, socket)
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
	err = runtime.SetupShutdownByDisconnect(context, receiver, "splitter.in", ch)
	if err != nil {
		log.Println("Failed to setup monitoring:", err.Error())
		os.Exit(1)
	}

	log.Println("Started")

	//  Process tasks forever
	for {
		log.Println("Waiting for IP...")
		ip, err := receiver.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			log.Println("Received invalid IP")
			continue
		}

		log.Println("Received IP:", string(ip[1]))
		for _, socket = range sendSockets {
			socket.SendMultipart(ip, 0)
		}
		log.Println("IP sent to all outputs")
	}
}
