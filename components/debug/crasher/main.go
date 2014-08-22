package main

import (
	"cascades/runtime"
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var (
	inputEndpoint  = flag.String("port.in", "", "Component's output port endpoint")
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

	context, _ := zmq.NewContext()
	defer context.Close()

	//  Socket to receive messages on
	receiver, _ := context.NewSocket(zmq.PULL)
	defer receiver.Close()
	receiver.Bind(*inputEndpoint)

	//  Socket to send messages to task sink
	sender, _ := context.NewSocket(zmq.PUSH)
	defer sender.Close()
	sender.Connect(*outputEndpoint)

	go func() {
		log.Println("Waiting for a crash")
		time.Sleep(3 * time.Second)
		sender.Close()
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	log.Println("Started")

	// Process tasks forever
	for {
		ip, _ := receiver.RecvMultipart(0)
		if !runtime.IsPacket(ip) {
			log.Println("Received invalid IP. Ignoring...")
			continue
		}
		sender.SendMultipart(ip, zmq.NOBLOCK)
	}

	time.Sleep(1e9) //  Give 0MQ time to deliver: one second ==  1e9 ns

	log.Println("Stopped")
}
