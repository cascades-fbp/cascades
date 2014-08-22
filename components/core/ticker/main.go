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
	intervalEndpoint = flag.String("port.interval", "", "Component's input port endpoint")
	outputEndpoint   = flag.String("port.out", "", "Component's output port endpoint")
	json             = flag.Bool("json", false, "Print component documentation in JSON")
	debug            = flag.Bool("debug", false, "Enable debug mode")
)

func main() {
	flag.Parse()

	if *json {
		doc, _ := registryEntry.JSON()
		fmt.Println(string(doc))
		os.Exit(0)
	}

	if *intervalEndpoint == "" {
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

	var (
		err                 error
		inSocket, outSocket *zmq.Socket
	)

	context, _ := zmq.NewContext()
	defer context.Close()

	inSocket, err = context.NewSocket(zmq.PULL)
	if err != nil {
		fmt.Println("Error creating socket:", err.Error())
		os.Exit(1)
	}
	defer inSocket.Close()
	err = inSocket.Bind(*intervalEndpoint)
	if err != nil {
		fmt.Println("Error binding socket:", err.Error())
		os.Exit(1)
	}

	outSocket, err = context.NewSocket(zmq.PUSH)
	if err != nil {
		fmt.Println("Error creating socket:", err.Error())
		os.Exit(1)
	}
	defer outSocket.Close()
	err = outSocket.Connect(*outputEndpoint)
	if err != nil {
		fmt.Println("Error connecting to socket:", err.Error())
		os.Exit(1)
	}

	log.Println("Started")

	var interval time.Duration

	// Loop just to ignore errors
	for {
		ip, err := inSocket.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving IP:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) || !runtime.IsPacket(ip) {
			continue
		}
		interval, err = time.ParseDuration(string(ip[1]))
		if err != nil {
			log.Println("Error parsing duration from IP:", err.Error())
			continue
		}
		break
	}

	// Create a ticker
	ticker := time.NewTicker(interval)
	log.Printf("Configured to tick with interval: %v", interval)

	// Close input port
	inSocket.Close()

	// Ctrl+C handling
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range ch {
			ticker.Stop()
			log.Println("Give 0MQ time to deliver before stopping...")
			time.Sleep(1e9)
			log.Println("Stopped")
			os.Exit(0)
		}
	}()

	// Send the ticks
	for v := range ticker.C {
		msg := fmt.Sprintf("%v", v.Unix())
		log.Println(msg)
		outSocket.SendMultipart(runtime.NewPacket([]byte(msg)), zmq.NOBLOCK)
	}
}
