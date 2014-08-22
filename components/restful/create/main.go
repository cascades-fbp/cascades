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
	typeEndpoint     = flag.String("port.type", "", "Component's content-type port endpoint")
	dsnEndpoint      = flag.String("port.dsn", "", "Component's content-type port endpoint")
	requiredEndpoint = flag.String("port.required", "", "Component's content-type port endpoint")
	uniqueEndpoint   = flag.String("port.unique", "", "Component's content-type port endpoint")
	inputEndpoint    = flag.String("port.in", "", "Component's content-type port endpoint")
	outputEndpoint   = flag.String("port.out", "", "Component's output port endpoint")
	errorEndpoint    = flag.String("port.error", "", "Component's output port endpoint")
	json             = flag.Bool("json", false, "Print component documentation in JSON")
	debug            = flag.Bool("debug", false, "Enable debug mode")
)

var (
	contentType, dsn string
	required, unique []string
)

var (
	typeSocket, dsnSocket, requiredSocket, uniqueSocket, inSocket, outSocket, errSocket *zmq.Socket
)

func main() {
	flag.Parse()

	if *json {
		doc, _ := registryEntry.JSON()
		fmt.Println(string(doc))
		os.Exit(0)
	}

	if *typeEndpoint == "" || *dsnEndpoint == "" || *inputEndpoint == "" || *outputEndpoint == "" || *errorEndpoint == "" {
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

	typeSocket, _ = context.NewSocket(zmq.PULL)
	defer typeSocket.Close()
	typeSocket.Bind(*typeEndpoint)

	dsnSocket, _ = context.NewSocket(zmq.PULL)
	defer dsnSocket.Close()
	dsnSocket.Bind(*typeEndpoint)

	if *requiredEndpoint != "" {
		requiredSocket, _ = context.NewSocket(zmq.PULL)
		defer requiredSocket.Close()
		requiredSocket.Bind(*typeEndpoint)
	}

	if *uniqueEndpoint != "" {
		uniqueSocket, _ = context.NewSocket(zmq.PULL)
		defer uniqueSocket.Close()
		uniqueSocket.Bind(*typeEndpoint)
	}

	inSocket, _ = context.NewSocket(zmq.PULL)
	defer inSocket.Close()
	inSocket.Bind(*inputEndpoint)

	outSocket, _ = context.NewSocket(zmq.PUSH)
	defer outSocket.Close()
	outSocket.Connect(*outputEndpoint)

	errSocket, _ = context.NewSocket(zmq.PUSH)
	defer errSocket.Close()
	errSocket.Connect(*outputEndpoint)

	pollItems := zmq.PollItems{
		zmq.PollItem{Socket: typeSocket, Events: zmq.POLLIN},
		zmq.PollItem{Socket: dsnSocket, Events: zmq.POLLIN},
	}
	if requiredSocket != nil {
		pollItems = append(pollItems, zmq.PollItem{Socket: requiredSocket, Events: zmq.POLLIN})
	}
	if uniqueSocket != nil {
		pollItems = append(pollItems, zmq.PollItem{Socket: uniqueSocket, Events: zmq.POLLIN})
	}

	log.Println("Waiting for configuration IPs/IIPs")
	var (
		ip  [][]byte
		err error
	)
	for {
		_, err = zmq.Poll(pollItems, -1)
		if err != nil {
			log.Println("Error polling ports:", err.Error())
			continue
		}
		switch {

		case pollItems[0].REvents&zmq.POLLIN != 0:
			ip, err = pollItems[0].Socket.RecvMultipart(0)
			if err != nil {
				log.Printf("Failed to receive data. Error: %s", err.Error())
				continue
			}
			if !runtime.IsValidIP(ip) {
				continue
			}
			contentType = string(ip[1])

		case pollItems[1].REvents&zmq.POLLIN != 0:
			ip, err = pollItems[1].Socket.RecvMultipart(0)
			if err != nil {
				log.Printf("Failed to receive data. Error: %s", err.Error())
				continue
			}
			if !runtime.IsValidIP(ip) {
				continue
			}
			dsn = string(ip[1])

		case len(pollItems) > 2 && pollItems[2].REvents&zmq.POLLIN != 0:
			ip, err = pollItems[1].Socket.RecvMultipart(0)
			if err != nil {
				log.Printf("Failed to receive data. Error: %s", err.Error())
				continue
			}
			if !runtime.IsValidIP(ip) {
				continue
			}
			tmp := strings.Replace(string(ip[1]), " ", ``, -1)
			required = strings.Split(tmp, ",")

		case len(pollItems) > 3 && pollItems[3].REvents&zmq.POLLIN != 0:
			ip, err = pollItems[1].Socket.RecvMultipart(0)
			if err != nil {
				log.Printf("Failed to receive data. Error: %s", err.Error())
				continue
			}
			if !runtime.IsValidIP(ip) {
				continue
			}
			tmp := strings.Replace(string(ip[1]), " ", ``, -1)
			unique = strings.Split(tmp, ",")

		}
	}

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

	// Monitoring setup
	err = runtime.SetupShutdownByDisconnect(context, inSocket, "restful-create.in", exitCh)
	if err != nil {
		log.Println("Failed to setup monitoring:", err.Error())
		os.Exit(1)
	}

	log.Println("Started")

	var connId, method, uri string
	for {
		ip, err := inSocket.RecvMultipart(0)
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
			method = ""
			uri = ""
		case runtime.IsPacket(ip):
			if connId == "" {
				connId = string(ip[1])
			} else if method == "" {
				method = string(ip[1])
			} else if uri == "" {
				uri = string(ip[1])
			}
		case runtime.IsCloseBracket(ip):
			// Substream received, process it and respond
			log.Printf("New request: %v, %v, %v", connId, method, uri)
			outSocket.SendMultipart(runtime.NewOpenBracket(), 0)
			outSocket.SendMultipart(runtime.NewPacket([]byte(connId)), 0)
			outSocket.SendMultipart(runtime.NewPacket([]byte("200")), 0)
			outSocket.SendMultipart(runtime.NewPacket([]byte("Hello, world!")), 0)
			outSocket.SendMultipart(runtime.NewCloseBracket(), 0)
		}
	}

}
