package main

import (
	"flag"
	zmq "github.com/alecthomas/gozmq"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	inputEndpoint  = flag.String("port.in", "", "Component's input port endpoint")
	resultEndpoint = flag.String("port.res", "", "GET result port endpoint")
	errorEndpoint  = flag.String("port.err", "", "GET error port endpoint")
	doc            = flag.Bool("doc", false, "Print component documentation")
)

func main() {
	flag.Parse()

	// print documentation and exit
	if *doc {
		log.Println(docstring)
		os.Exit(0)
	}
	if *inputEndpoint == "" || *resultEndpoint == "" || *errorEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.SetFlags(0)

	context, _ := zmq.NewContext()
	defer context.Close()

	// Input socket
	inSocket, _ := context.NewSocket(zmq.PULL)
	defer inSocket.Close()
	inSocket.Bind(*inputEndpoint)

	// Result output socket
	resSocket, _ := context.NewSocket(zmq.PUSH)
	defer resSocket.Close()
	resSocket.Connect(*resultEndpoint)

	// Error output socket
	errSocket, _ := context.NewSocket(zmq.PUSH)
	defer errSocket.Close()
	errSocket.Connect(*errorEndpoint)

	log.Println("Started")

	// Ctrl+C handling
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range ch {
			// close 0mq sockets
			inSocket.Close()
			errSocket.Close()
			resSocket.Close()

			log.Println("Stopped")
			os.Exit(0)
		}
	}()

	for {
		url, _ := inSocket.Recv(0)
		log.Println("Received URL to process", string(url))

		resp, err := http.Get(string(url))
		if err != nil {
			// Send to ERR: [url 1 err_string]
			errmsg := [][]byte{url, []byte{1}, []byte(err.Error())}
			if e := errSocket.SendMultipart(errmsg, zmq.NOBLOCK); e != nil {
				log.Fatal("Error sending:", e)
			}
		} else {
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)

			if resp.StatusCode != 200 {
				// Send to ERR: [url code body]
				errmsg := [][]byte{url, []byte{byte(resp.StatusCode)}, body}
				if e := errSocket.SendMultipart(errmsg, zmq.NOBLOCK); e != nil {
					log.Fatal("Error sending:", e)
				}
			} else {
				// Send to RES: [url body]
				resmsg := [][]byte{url, body}
				if e := resSocket.SendMultipart(resmsg, zmq.NOBLOCK); e != nil {
					log.Fatal("Error sending:", e)
				}
			}
		}
	}
}
