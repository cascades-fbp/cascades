package main

import (
	"cascades/components/http/utils"
	"cascades/runtime"
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	// Data from http handler and data to http handler
	inCh := make(chan utils.HTTPResponse)
	outCh := make(chan HandlerRequest)
	go func(ctx *zmq.Context, endpoint string) {
		// Socket to send messages to task sink
		sender, _ := context.NewSocket(zmq.PUSH)
		defer sender.Close()
		sender.Connect(endpoint)
		// Map of uuid to requests
		dataMap := make(map[string]chan utils.HTTPResponse)
		// Start listening in/out channels
		for {
			select {
			case data := <-outCh:
				dataMap[data.Request.Id] = data.ResponseCh
				ip, _ := utils.Request2IP(data.Request)
				sender.SendMultipart(ip, 0)
			case resp := <-inCh:
				if respCh, ok := dataMap[resp.Id]; ok {
					log.Println("Resolved channel for response", resp.Id)
					respCh <- resp
					delete(dataMap, resp.Id)
					continue
				}
				log.Println("Didn't find request handler mapping for a given ID", resp.Id)
			}
		}
	}(context, *outputEndpoint)

	// Web server goroutine
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", Handler(outCh))

		s := &http.Server{
			Handler:        mux,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		ln, err := net.Listen("tcp", bindAddr)
		if err != nil {
			log.Println(err.Error())
			exitCh <- syscall.SIGTERM
			return
		}

		log.Printf("Starting listening %v", bindAddr)
		err = s.Serve(ln)
		if err != nil {
			log.Println(err.Error())
			exitCh <- syscall.SIGTERM
			return
		}
	}()

	// Process incoming message forever
	for {
		ip, err := receiver.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			log.Println("Received invalid IP")
			continue
		}

		resp, err := utils.IP2Response(ip)
		if err != nil {
			log.Printf("Error converting IP to response: %s", err.Error())
			continue
		}
		inCh <- *resp
	}
}
