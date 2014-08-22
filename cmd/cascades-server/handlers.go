package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"log"
	"net/http"
	"time"
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

func GetRuntimesHandler(w http.ResponseWriter, r *http.Request) {
	runtime := struct {
		Id          string    `json:"id"`
		User        string    `json:"user"`
		Secret      string    `json:"secret"`
		Address     string    `json:"address"`
		Protocol    string    `json:"protocol"`
		Type        string    `json:"type"`
		Label       string    `json:"label"`
		Description string    `json:"description"`
		Registered  time.Time `json:"registered"`
		Seen        time.Time `json:"seen"`
	}{
		Id:          "35907A5D-ADF7-42F9-85E0-E339F690204B",
		User:        "06594d47-5435-45a3-bdf4-c1a3372f2824",
		Secret:      "FASds8s98asfb",
		Address:     "ws://127.0.0.1:3000/runtime",
		Protocol:    "websocket",
		Type:        "custom",
		Label:       "Local Cascades Runtime",
		Description: "Cascades Runtime - A polyglot ZMQ-based DataFlow/FBP platform",
		Registered:  time.Now(),
		Seen:        time.Now(),
	}
	runtimes := []interface{}{runtime}
	content, err := json.Marshal(runtimes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}
