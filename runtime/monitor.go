package runtime

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	zmq "github.com/pebbe/zmq4"
)

const MONITOR_EVENTS zmq.Event = zmq.EVENT_CONNECTED | zmq.EVENT_LISTENING |
	zmq.EVENT_ACCEPTED | zmq.EVENT_BIND_FAILED | zmq.EVENT_ACCEPT_FAILED | zmq.EVENT_CLOSED |
	zmq.EVENT_DISCONNECTED

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
			eventId := zmq.Event(binary.LittleEndian.Uint16(data[0][:2]))
			switch eventId {
			case zmq.EVENT_CONNECTED:
				log.Println("EVENT_CONNECTED", string(data[1]))
			case zmq.EVENT_CONNECT_DELAYED:
				log.Println("EVENT_CONNECT_DELAYED", string(data[1]))
			case zmq.EVENT_CONNECT_RETRIED:
				log.Println("EVENT_CONNECT_RETRIED", string(data[1]))
			case zmq.EVENT_LISTENING:
				log.Println("EVENT_LISTENING", string(data[1]))
			case zmq.EVENT_BIND_FAILED:
				log.Println("EVENT_BIND_FAILED", string(data[1]))
			case zmq.EVENT_ACCEPTED:
				log.Println("EVENT_ACCEPTED", string(data[1]))
			case zmq.EVENT_ACCEPT_FAILED:
				log.Println("EVENT_ACCEPT_FAILED", string(data[1]))
			case zmq.EVENT_CLOSED:
				log.Println("EVENT_CLOSED", string(data[1]))
			case zmq.EVENT_CLOSE_FAILED:
				log.Println("EVENT_CLOSE_FAILED", string(data[1]))
			case zmq.EVENT_DISCONNECTED:
				log.Println("EVENT_DISCONNECTED", string(data[1]))
			default:
				log.Printf("Unsupported event id: %#v - Message: %#v", eventId, data)
			}
			monCh <- zmq.Event(eventId)
		}
	}()
	return monCh, socket.Monitor(endpoint, MONITOR_EVENTS)
}

//
// This function is a helper shortcut to setup shutdown behavior once an accepted connection
// closes (disconnects).
//
func SetupShutdownByDisconnect(socket *zmq.Socket, name string, termChannel chan os.Signal) error {
	// Monitoring setup
	ch, err := MonitorSocket(socket, name)
	if err != nil {
		return err
	}
	go func() {
		connections := 0
		for e := range ch {
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
			if connections == 0 {
				log.Println("No connections. Sending termination signal...")
				termChannel <- syscall.SIGTERM
				break
			}
		}
	}()

	return nil
}
