package main

import (
	"bufio"
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
	inputEndpoint  = flag.String("port.file", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	errorEndpoint  = flag.String("port.err", "", "Component's error port endpoint")
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

	var errorSocket *zmq.Socket
	if *outputEndpoint != "" {
		errorSocket, _ = context.NewSocket(zmq.PUSH)
		defer errorSocket.Close()
		errorSocket.Connect(*errorEndpoint)
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
	err = runtime.SetupShutdownByDisconnect(context, receiver, "passthru.in", ch)
	if err != nil {
		log.Println("Failed to setup monitoring:", err.Error())
		os.Exit(1)
	}

	log.Println("Started")

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

		filepath := string(ip[1])
		f, err := os.Open(filepath)
		if err != nil {
			log.Printf("ERROR openning file %s: %s", filepath, err.Error())
			if errorSocket != nil {
				errorSocket.SendMultipart(runtime.NewPacket([]byte(err.Error())), zmq.NOBLOCK)
			}
			continue
		}

		sender.SendMultipart(runtime.NewOpenBracket(), 0)
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			sender.SendMultipart(runtime.NewPacket(scanner.Bytes()), 0)
		}
		if err = scanner.Err(); err != nil {
			errorSocket.SendMultipart(runtime.NewPacket([]byte(err.Error())), zmq.NOBLOCK)
		}
		f.Close()
		sender.SendMultipart(runtime.NewCloseBracket(), 0)

		sender.SendMultipart(ip, zmq.NOBLOCK)
	}
}
