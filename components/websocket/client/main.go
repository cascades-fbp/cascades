package main

import (
	"code.google.com/p/go.net/websocket"
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	urlString      = flag.String("meta.url", "ws://127.0.0.1:8080/", "WebSocket server's url")
	inputEndpoint  = flag.String("port.in", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
)

type Connection struct {
	WS      *websocket.Conn
	Send    chan []byte
	Receive chan []byte
}

func NewConnection(wsConn *websocket.Conn) *Connection {
	connection := &Connection{
		WS:      wsConn,
		Send:    make(chan []byte, 256),
		Receive: make(chan []byte, 256),
	}
	return connection
}

func (self *Connection) Reader() {
	for {
		var data []byte
		err := websocket.Message.Receive(self.WS, &data)
		if err != nil {
			log.Println("Reader: error receiving message")
			continue
		}
		log.Println("Reader: received from websocket", data)
		self.Receive <- data
	}
	self.WS.Close()
}

func (self *Connection) Writer() {
	for data := range self.Send {
		log.Println("Writer: will send to websocket", data)
		err := websocket.Message.Send(self.WS, data)
		if err != nil {
			break
		}
	}
	self.WS.Close()
}

func (self *Connection) Close() {
	close(self.Send)
	close(self.Receive)
	self.WS.Close()
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	flag.Parse()

	_, err := url.Parse(*urlString)
	if err != nil {
		log.Println("Invalid url provided.", err.Error())
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

	// Establish WS connection
	hostname, _ := os.Hostname()
	origin := fmt.Sprintf("http://%s", hostname)
	ws, err := websocket.Dial(*urlString, "", origin)
	if err != nil {
		log.Println("Failed to dial server.", err.Error())
		os.Exit(1)
	}
	connection := NewConnection(ws)
	defer func() {
		connection.Close()
	}()
	go connection.Reader()
	go connection.Writer()
	go func(socket *zmq.Socket) {
		for data := range connection.Receive {
			log.Println("Sending data from websocket to OUT port...")
			socket.Send(data, zmq.NOBLOCK)
		}
	}(sender)

	log.Println("Started")

	// Listen to packets from IN port
	for {
		parts, err := receiver.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}

		log.Println("Sending data from IN port to websocket...")

		// Send data to WS connection
		for _, p := range parts {
			connection.Send <- p
		}
	}
}
