package utils

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	zmq "github.com/pebbe/zmq4"
)

// MonitorEvents is a bit mask of ZMQ events to listen to
const MonitorEvents zmq.Event = zmq.EVENT_CONNECTED | zmq.EVENT_LISTENING |
	zmq.EVENT_ACCEPTED | zmq.EVENT_BIND_FAILED | zmq.EVENT_ACCEPT_FAILED | zmq.EVENT_CLOSED |
	zmq.EVENT_DISCONNECTED

// AssertError prints given error message if err is not nil & exist with status code 1
func AssertError(err error) {
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}
}

// CreateInputPort creates a ZMQ PULL socket & bind to a given endpoint
func CreateInputPort(endpoint string) (socket *zmq.Socket, err error) {
	return CreateMonitoredInputPort("", endpoint, nil)
}

// CreateMonitoredInputPort creates a ZMQ PULL socket & bind to a given endpoint
func CreateMonitoredInputPort(name string, endpoint string, termChannel chan os.Signal) (socket *zmq.Socket, err error) {
	socket, err = zmq.NewSocket(zmq.PULL)
	if err != nil {
		return nil, err
	}
	if termChannel == nil {
		return socket, socket.Bind(endpoint)
	}

	ch, err := MonitorSocket(socket, name)
	if err != nil {
		return nil, err
	}
	err = socket.Bind(endpoint)
	if err != nil {
		return nil, err
	}

	log.Println("Waiting for input port connection to establish... ")
	for connected := false; !connected; {
		select {
		case e := <-ch:
			if e == zmq.EVENT_ACCEPTED {
				log.Println("Input port connected (EVENT_ACCEPTED)")
				connected = true
			}
			//log.Println(">> IN:", e)
		case <-time.Tick(30 * time.Second):
			return nil, fmt.Errorf("Timeout: input port connection was not established within provided interval")
		}
	}

	go shutdownByDisconnect(ch, termChannel)

	return socket, nil
}

// CreateOutputPort creates a ZMQ PUSH socket & connect to a given endpoint
func CreateOutputPort(endpoint string) (socket *zmq.Socket, err error) {
	return CreateMonitoredOutputPort("", endpoint, nil)
}

// CreateMonitoredOutputPort creates a ZMQ PUSH socket & connect to a given endpoint
func CreateMonitoredOutputPort(name string, endpoint string, termChannel chan os.Signal) (socket *zmq.Socket, err error) {
	socket, err = zmq.NewSocket(zmq.PUSH)
	if err != nil {
		return nil, err
	}
	if termChannel == nil {
		return socket, socket.Connect(endpoint)
	}

	ch, err := MonitorSocket(socket, name)
	if err != nil {
		return nil, err
	}
	err = socket.Connect(endpoint)
	if err != nil {
		return nil, err
	}

	log.Println("Waiting for output port connection to establish... ")
	for connected := false; !connected; {
		select {
		case e := <-ch:
			if e == zmq.EVENT_CONNECTED {
				log.Println("Output port connected (EVENT_CONNECTED)")
				connected = true
			}
			//log.Println(">> OUT:", e)
		case <-time.Tick(30 * time.Second):
			return nil, fmt.Errorf("Timeout: output port connection was not established within provided interval")
		}
	}

	go shutdownByDisconnect(ch, termChannel)

	return socket, nil
}

//
// SetupShutdownByDisconnect is a helper shortcut to setup shutdown behavior once an accepted connection
// closes (disconnects).
//
func shutdownByDisconnect(eventChannel <-chan zmq.Event, termChannel chan os.Signal) {
	connections := 0
	for e := range eventChannel {
		switch e {
		case zmq.EVENT_ACCEPTED:
			connections++
			log.Println("Accepted connection to a socket. Total number of connections:", connections)
		case zmq.EVENT_DISCONNECTED:
			connections--
			log.Println("Client disconnected from a socket. Total number of connections:", connections)
		case zmq.EVENT_CLOSED:
			connections--
			log.Println("Connection's underlying descriptor has been closed. Total number of connections:", connections)
		}
		if connections <= 0 {
			log.Println("No connections. Sending termination signal...")
			termChannel <- syscall.SIGTERM
			break
		}
	}
}

//
// MonitorSocket creates a monitoring socket using given context and connects
// to a given socket to be monitored. Returns a channel to receive monitoring
// events. See event definitions here: http://api.zeromq.org/3-2:zmq-socket-monitor
//
func MonitorSocket(socket *zmq.Socket, name string) (<-chan zmq.Event, error) {
	endpoint := fmt.Sprintf("inproc://%v.%v.%v", name, os.Getpid(), time.Now().UnixNano())
	monCh := make(chan zmq.Event, 512) // make a buffered channel in case of heavy network activity
	go func() {
		monSock, err := zmq.NewSocket(zmq.PAIR)
		if err != nil {
			log.Println("Failed to start monitoring socket:", err.Error())
			return
		}
		monSock.Connect(endpoint)
		for {
			data, err := monSock.RecvMessageBytes(0)
			if err != nil {
				log.Println("Error receiving monitoring message:", err.Error())
				return
			}
			eventID := zmq.Event(binary.LittleEndian.Uint16(data[0][:2]))
			/*
				switch eventID {
				case zmq.EVENT_CONNECTED:
					log.Println("MonitorSocket: EVENT_CONNECTED", string(data[1]))
				case zmq.EVENT_CONNECT_DELAYED:
					log.Println("MonitorSocket: EVENT_CONNECT_DELAYED", string(data[1]))
				case zmq.EVENT_CONNECT_RETRIED:
					log.Println("MonitorSocket: EVENT_CONNECT_RETRIED", string(data[1]))
				case zmq.EVENT_LISTENING:
					log.Println("MonitorSocket: EVENT_LISTENING", string(data[1]))
				case zmq.EVENT_BIND_FAILED:
					log.Println("MonitorSocket: EVENT_BIND_FAILED", string(data[1]))
				case zmq.EVENT_ACCEPTED:
					log.Println("MonitorSocket: EVENT_ACCEPTED", string(data[1]))
				case zmq.EVENT_ACCEPT_FAILED:
					log.Println("MonitorSocket: EVENT_ACCEPT_FAILED", string(data[1]))
				case zmq.EVENT_CLOSED:
					log.Println("MonitorSocket: EVENT_CLOSED", string(data[1]))
				case zmq.EVENT_CLOSE_FAILED:
					log.Println("MonitorSocket: EVENT_CLOSE_FAILED", string(data[1]))
				case zmq.EVENT_DISCONNECTED:
					log.Println("MonitorSocket: EVENT_DISCONNECTED", string(data[1]))
				default:
					log.Printf("MonitorSocket: Unsupported event id: %#v - Message: %#v", eventID, data)
				}
			*/
			monCh <- zmq.Event(eventID)
		}
	}()
	return monCh, socket.Monitor(endpoint, MonitorEvents)
}
