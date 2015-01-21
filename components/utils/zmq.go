package utils

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
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
func CreateInputPort(name string, endpoint string, monitCh chan<- bool) (socket *zmq.Socket, err error) {
	socket, err = zmq.NewSocket(zmq.PULL)
	if err != nil {
		return nil, err
	}
	if monitCh == nil {
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

	go func() {
		c := 0
		for e := range ch {
			if e == zmq.EVENT_ACCEPTED {
				c++
				if c == 1 {
					monitCh <- true
				}
			} else if e == zmq.EVENT_CLOSED || e == zmq.EVENT_DISCONNECTED {
				c--
				if c == 0 {
					monitCh <- false
				}
			}
			if c < 0 {
				c = 0
			}
		}
	}()

	return socket, nil
}

// CreateOutputPort creates a ZMQ PUSH socket & connect to a given endpoint
func CreateOutputPort(name string, endpoint string, monitCh chan<- bool) (socket *zmq.Socket, err error) {
	socket, err = zmq.NewSocket(zmq.PUSH)
	if err != nil {
		return nil, err
	}
	if monitCh == nil {
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

	go func() {
		c := 0
		for e := range ch {
			if e == zmq.EVENT_ACCEPTED || e == zmq.EVENT_CONNECTED {
				c++
				if c == 1 {
					monitCh <- true
				}
			} else if e == zmq.EVENT_CLOSED || e == zmq.EVENT_DISCONNECTED {
				c--
				if c == 0 {
					monitCh <- false
				}
			}
			if c < 0 {
				c = 0
			}
		}
	}()

	return socket, nil
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
