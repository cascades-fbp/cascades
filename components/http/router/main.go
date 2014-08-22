package main

import (
	"cascades/components/http/utils"
	"cascades/runtime"
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	patternEndpoint = flag.String("port.pattern", "", "Component's input port endpoint")
	requestEndpoint = flag.String("port.request", "", "Component's input port endpoint")
	successEndpoint = flag.String("port.success", "", "Component's output port endpoint")
	failEndpoint    = flag.String("port.fail", "", "Component's output port endpoint")
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

	if *patternEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *requestEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *successEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *failEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.SetFlags(0)
	if *debug {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	// Check pattern->success arrays correspondance
	var (
		patterns  []string
		successes []string
	)
	patterns = strings.Split(*patternEndpoint, ",")
	successes = strings.Split(*successEndpoint, ",")
	if len(patterns) != len(successes) {
		fmt.Println("ERROR: the length of PATTERN array port should be same as length of SUCCESS array port!")
		flag.Usage()
		os.Exit(1)
	}

	// ZMQ context
	context, _ := zmq.NewContext()
	defer context.Close()

	// Create pattern/success sockets
	var (
		socket    *zmq.Socket
		err       error
		pollItems = zmq.PollItems{}
	)
	patternSockets := []*zmq.Socket{}
	successSockets := []*zmq.Socket{}
	for i, endpoint := range patterns {
		// Add pattern IN socket
		socket, err = context.NewSocket(zmq.PULL)
		if err != nil {
			fmt.Println("Error creating socket:", err.Error())
			os.Exit(1)
		}
		defer socket.Close()
		endpoint = strings.TrimSpace(endpoint)
		err = socket.Bind(endpoint)
		if err != nil {
			fmt.Println("Error binding socket:", err.Error())
			os.Exit(1)
		}
		patternSockets = append(patternSockets, socket)

		// Add pattens to poll items
		pollItems = append(pollItems, zmq.PollItem{Socket: socket, Events: zmq.POLLIN})

		// Add success OUT socket
		socket, err = context.NewSocket(zmq.PUSH)
		if err != nil {
			fmt.Println("Error creating socket:", err.Error())
			os.Exit(1)
		}
		defer socket.Close()
		endpoint = strings.TrimSpace(successes[i])
		err = socket.Connect(endpoint)
		if err != nil {
			fmt.Println("Error connecting to socket:", err.Error())
			os.Exit(1)
		}
		successSockets = append(successSockets, socket)
	}

	// Create/bind data in socket
	reqSocket, err := context.NewSocket(zmq.PULL)
	if err != nil {
		fmt.Println("Error creating socket:", err.Error())
		os.Exit(1)
	}
	defer reqSocket.Close()
	err = reqSocket.Bind(*requestEndpoint)
	if err != nil {
		fmt.Println("Error binding socket:", err.Error())
		os.Exit(1)
	}
	pollItems = append(pollItems, zmq.PollItem{Socket: reqSocket, Events: zmq.POLLIN})

	// Create/connect fail out socket
	failSock, err := context.NewSocket(zmq.PUSH)
	if err != nil {
		fmt.Println("Error creating socket:", err.Error())
		os.Exit(1)
	}
	defer failSock.Close()
	err = failSock.Connect(*failEndpoint)
	if err != nil {
		fmt.Println("Error connecting to socket:", err.Error())
		os.Exit(1)
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

	// Monitoring of request IN socket
	err = runtime.SetupShutdownByDisconnect(context, reqSocket, "http-router.in", exitCh)
	if err != nil {
		log.Println("Failed to setup monitoring:", err.Error())
		os.Exit(1)
	}

	// Main loop
	var (
		index       int = -1
		outputIndex int = -1
		params      url.Values
		ip          [][]byte
		pLength     int     = len(pollItems)
		router      *Router = NewRouter()
	)
	for {
		// Poll sockets
		log.Println("Polling sockets...")

		_, err = zmq.Poll(pollItems, -1)
		if err != nil {
			log.Println("Error polling ports:", err.Error())
			os.Exit(1)
		}

		// Resolve socket index
		for i, item := range pollItems {
			if item.REvents&zmq.POLLIN != 0 {
				index = i
				break
			}
		}

		ip, err = pollItems[index].Socket.RecvMultipart(0)
		if !runtime.IsValidIP(ip) {
			log.Println("Received invalid IP")
			continue
		}
		if err != nil {
			log.Printf("Failed to receive data. Error: %s", err.Error())
			continue
		}

		// Pattern arrived
		if index < pLength-1 {
			// Close pattern socket
			socket = pollItems[index].Socket
			socket.Close()

			// Resolve corresponding output socket index
			outputIndex = -1
			for i, s := range patternSockets {
				if s == socket {
					outputIndex = i
				}
			}
			if outputIndex == -1 {
				log.Printf("Failed to resolve output socket index")
				continue
			}

			// Remove closed socket from polling items
			pollItems = append(pollItems[:index], pollItems[index+1:]...)
			pLength -= 1

			// Add pattern to router
			parts := strings.Split(string(ip[1]), " ")
			method := strings.ToUpper(strings.TrimSpace(parts[0]))
			pattern := strings.TrimSpace(parts[1])
			switch method {
			case "GET":
				router.Get(pattern, outputIndex)
			case "POST":
				router.Post(pattern, outputIndex)
			case "PUT":
				router.Put(pattern, outputIndex)
			case "DELETE":
				router.Del(pattern, outputIndex)
			case "HEAD":
				router.Head(pattern, outputIndex)
			case "OPTIONS":
				router.Options(pattern, outputIndex)
			default:
				log.Printf("Unsupported HTTP method %s in pattern %s", method, pattern)
			}
			continue
		}

		// Request arrive
		req, err := utils.IP2Request(ip)
		if err != nil {
			log.Printf("Failed to convert IP to request. Error: %s", err.Error())
			continue
		}

		outputIndex, params = router.Route(req.Method, req.URI)
		log.Printf("Output index for %s %s: %v (params=%#v)", req.Method, req.URI, outputIndex, params)

		switch outputIndex {
		case NotFound:
			log.Println("Sending Not Found response to FAIL output")
			resp := &utils.HTTPResponse{
				Id:         req.Id,
				StatusCode: http.StatusNotFound,
			}
			ip, _ = utils.Response2IP(resp)
			failSock.SendMultipart(ip, 0)
		case MethodNotAllowed:
			log.Println("Sending Method Not Allowed response to FAIL output")
			resp := &utils.HTTPResponse{
				Id:         req.Id,
				StatusCode: http.StatusMethodNotAllowed,
			}
			ip, _ = utils.Response2IP(resp)
			failSock.SendMultipart(ip, 0)
		default:
			for k, values := range params {
				req.Form[k] = values
			}
			ip, _ = utils.Request2IP(req)
			successSockets[outputIndex].SendMultipart(ip, 0)
		}

		index = -1
	}
}
