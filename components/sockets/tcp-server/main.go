package main

import (
	"cascades/runtime"
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"io/ioutil"
	"log"
	"net"
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

	//  Socket to receive messages on
	receiver, _ := context.NewSocket(zmq.PULL)
	defer receiver.Close()
	receiver.Bind(*inputEndpoint)

	// Create service
	service := NewService()

	// Ctrl+C handling
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range ch {
			service.Stop()
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

	// Create binding address listener
	laddr, err := net.ResolveTCPAddr("tcp", bindAddr)
	if err != nil {
		log.Fatalln(err)
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Listening on", listener.Addr())

	// Start server with listener
	go service.Serve(listener)
	go func() {
		//  Socket to send messages
		sender, _ := context.NewSocket(zmq.PUSH)
		defer sender.Close()
		sender.Connect(*outputEndpoint)
		for data := range service.Output {
			sender.SendMultipart(runtime.NewOpenBracket(), 0)
			sender.SendMultipart(runtime.NewPacket(data[0]), 0)
			sender.SendMultipart(runtime.NewPacket(data[1]), 0)
			sender.SendMultipart(runtime.NewCloseBracket(), 0)
		}
	}()

	//  Process tasks forever
	var (
		connId string
		data   []byte
	)
	for {
		ip, err := receiver.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			continue
		}
		switch {
		case runtime.IsOpenBracket(ip):
			connId = ""
			data = nil
		case runtime.IsPacket(ip):
			if connId == "" {
				connId = string(ip[1])
			} else {
				data = ip[1]
			}
		case runtime.IsCloseBracket(ip):
			service.Dispatch(connId, data)
		}
	}
}
