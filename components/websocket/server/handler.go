package main

import (
	"code.google.com/p/go.net/websocket"
	"log"
)

func WebHandler(ws *websocket.Conn) {
	log.Printf("WebHandler: New connection from %v", ws.RemoteAddr())

	// Create a new connection structure
	c := NewConnection(ws)

	// Notify hub to register the new connection
	DefaultHub.Register <- c

	// Defer the hub notification to unregister when this connection is closed
	defer func() {
		log.Printf("WebHandler: deferred call to remove connection from registry %v", ws.RemoteAddr())
		DefaultHub.Unregister <- c
	}()

	// Start writing in a separate goroutine
	go c.Writer()

	// Start reading from socket in the current goroutine
	c.Reader()
}
