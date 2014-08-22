package main

import (
	uuid "github.com/nu7hatch/gouuid"
	"net"
)

type Connection struct {
	Id      string
	TCPConn *net.TCPConn
	Input   chan []byte
}

func NewConnection(tcpConn *net.TCPConn, in chan []byte) *Connection {
	id, _ := uuid.NewV4()
	c := &Connection{
		Id:      id.String(),
		TCPConn: tcpConn,
		Input:   in,
	}
	return c
}

func (self *Connection) Close() {
	self.TCPConn.Close()
	close(self.Input)
}
