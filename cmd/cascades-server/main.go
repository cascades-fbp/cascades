package main

import (
	"cascades/registry"
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
)

var (
	bindAddr      = flag.String("bind", "0.0.0.0", "HTTP address to bind (default 0.0.0.0)")
	httpPort      = flag.Int("port", 3000, "HTTP port to listen (default 3000)")
	staticDir     = flag.String("static", "www", "Web-server static document root folder")
	indexFilepath = flag.String("index", "conf/registry.json", "File path to JSON components index")
)

var (
	componentsDb registry.JSONRegistry
)

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[cascades] ")

	// parse index if exists
	data, err := ioutil.ReadFile(*indexFilepath)
	if err != nil {
		log.Println("Failed to read existing index file:" + err.Error())
		return
	}
	err = json.Unmarshal(data, &componentsDb)
	if err != nil {
		log.Println("Failed to parse index file:" + err.Error())
		return
	}

	// Start signal catching routine for a propert exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func(ch chan os.Signal) {
		s := <-c
		log.Println("Got signal:", s)
		os.Exit(0)
	}(c)

	http.Handle("/", http.FileServer(http.Dir(*staticDir)))
	http.Handle("/runtime", websocket.Handler(WebHandler))
	http.HandleFunc("/noflo/runtimes/", GetRuntimesHandler)

	// Start connections hub
	go DefaultHub.Start()

	// Listen & serve
	addr := fmt.Sprintf("%v:%v", *bindAddr, *httpPort)
	log.Printf("Listening %v", addr)
	log.Println("Serving static from:", *staticDir)
	log.Printf("Web-socket endpoint: ws://%s:%v/ws", *bindAddr, *httpPort)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe Error:", err)
	}

}
