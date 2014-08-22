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

	// Socket to send messages On
	sender, _ := context.NewSocket(zmq.PUSH)
	defer sender.Close()
	sender.Connect(*outputEndpoint)

	delay := 5 * time.Second
	log.Printf("Waiting before sending for %v", delay)
	time.Sleep(delay)

	log.Println("Sending test IP...")
	sender.SendMultipart(runtime.NewPacket([]byte("This is my last will")), 0)

	log.Println("Give 0MQ time to deliver before stopping...")
	time.Sleep(1e9)
	log.Println("Stopped")
	os.Exit(0)
}
