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
	"strings"
	"syscall"
	"time"
)

var (
	inputEndpoint  = flag.String("port.in", "", "Component's input port #1 endpoint")
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

	inports := strings.Split(*inputEndpoint, ",")
	if len(inports) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var err error
	context, _ := zmq.NewContext()
	defer context.Close()

	//  Socket to send messages
	var (
		socket *zmq.Socket
	)
	receiveSockets := []*zmq.Socket{}
	pollItems := zmq.PollItems{}

	for i, endpoint := range inports {
		// create socket
		socket, err = context.NewSocket(zmq.PULL)
		if err != nil {
			fmt.Println("Error creating socket:", err.Error())
			os.Exit(1)
		}
		defer socket.Close()

		// bind to endpoint
		endpoint = strings.TrimSpace(endpoint)
		log.Printf("Binding OUT[%v]=%s", i, endpoint)
		err = socket.Bind(endpoint)
		if err != nil {
			fmt.Println("Error binding socket:", err.Error())
			os.Exit(1)
		}
		receiveSockets = append(receiveSockets, socket)

		// add to polling items
		pollItems = append(pollItems, zmq.PollItem{Socket: socket, Events: zmq.POLLIN})
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

	log.Println("Started")

	var (
		ip [][]byte
	)
	for {
		_, err = zmq.Poll(pollItems, -1)
		if err != nil {
			log.Println("Error polling ports:", err.Error())
			continue
		}

		socket = nil
		for _, item := range pollItems {
			if item.REvents&zmq.POLLIN != 0 {
				socket = item.Socket
				break
			}
		}
		if socket == nil {
			log.Println("ERROR: could not find socket in polling items array")
			continue
		}

		ip, err = socket.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			log.Println("Received invalid IP")
			continue
		}

		sender.SendMultipart(ip, 0)
	}
}
