package main

import (
	"bytes"
	"log"
	"net"
	"sync"
	"time"
)

type Service struct {
	done      chan bool
	waitGroup *sync.WaitGroup
	dataMap   map[string]*Connection
	Output    chan [][]byte
}

// Service constructor
func NewService() *Service {
	s := &Service{
		done:      make(chan bool),
		waitGroup: &sync.WaitGroup{},
		dataMap:   make(map[string]*Connection),
		Output:    make(chan [][]byte),
	}
	s.waitGroup.Add(1)
	return s
}

// Accept connections and spawn a goroutine to serve each one.  Stop listening
// if anything is received on the service's channel.
func (self *Service) Serve(listener *net.TCPListener) {
	defer self.waitGroup.Done()
	for {
		select {
		case <-self.done:
			log.Println("Stopping listening on", listener.Addr())
			listener.Close()
			return
		default:
		}

		listener.SetDeadline(time.Now().Add(1e9))
		conn, err := listener.AcceptTCP()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			log.Println(err)
		}

		log.Println(conn.RemoteAddr(), "connected")

		connection := NewConnection(conn, make(chan []byte))
		self.dataMap[connection.Id] = connection
		self.waitGroup.Add(1)
		go self.serve(connection)
	}
}

func (self *Service) Dispatch(id string, data []byte) {
	if conn, ok := self.dataMap[id]; ok {
		conn.Input <- data
		return
	}
	log.Printf("ERROR: Did not find data channel for connection %s", id)
}

// Stop the service by closing the service's channel.  Block until the service
// is really stopped.
func (self *Service) Stop() {
	close(self.Output)
	close(self.done)
	self.waitGroup.Wait()
}

// Serve a connection by reading and writing what was read.  That's right, this
// is an echo service.  Stop reading and writing if anything is received on the
// service's channel but only after writing what was read.
func (self *Service) serve(connection *Connection) {
	defer connection.Close()
	defer self.waitGroup.Done()
	for {
		select {
		case <-self.done:
			log.Println("Disconnecting", connection.TCPConn.RemoteAddr())
			return
		default:
		}

		connection.TCPConn.SetDeadline(time.Now().Add(30 * time.Second))
		buf := make([]byte, 4096)
		if _, err := connection.TCPConn.Read(buf); err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			log.Println("Error reading from connection:", err)
			delete(self.dataMap, connection.Id)
			break
		}

		payload := bytes.TrimRight(buf, string([]byte{0x00, '\n', '\r'}))
		self.Output <- [][]byte{[]byte(connection.Id), payload}

		data := <-connection.Input
		data = append(data, '\n', '\r')
		if _, err := connection.TCPConn.Write(data); err != nil {
			log.Println(err)
			break
		}
	}
}
