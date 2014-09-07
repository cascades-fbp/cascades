package main

import (
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/cascades-fbp/cascades/runtime"
	"github.com/howeyc/fsnotify"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var (
	inputEndpoint  = flag.String("port.dir", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.created", "", "Component's output port endpoint")
	errorEndpoint  = flag.String("port.err", "", "Component's error port endpoint")
	json           = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")
)

func assertError(err error) {
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func main() {
	flag.Parse()

	if *json {
		doc, _ := registryEntry.JSON()
		fmt.Println(string(doc))
		os.Exit(0)
	}

	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *outputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.SetFlags(0)
	if *debug {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	var err error
	context, _ := zmq.NewContext()
	defer context.Close()

	//  Socket to receive messages on
	receiver, err := context.NewSocket(zmq.PULL)
	assertError(err)
	defer receiver.Close()
	err = receiver.Bind(*inputEndpoint)
	assertError(err)

	var errorSocket *zmq.Socket
	if *errorEndpoint != "" {
		errorSocket, _ = context.NewSocket(zmq.PUSH)
		defer errorSocket.Close()
		errorSocket.Connect(*errorEndpoint)
	}

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

	//TODO: monitor for input port, close socket when disconnected

	// Setup watcher
	watcher, err := fsnotify.NewWatcher()
	assertError(err)
	defer watcher.Close()

	// Process events
	go func() {
		//  Socket to send messages to task sink
		sender, err := context.NewSocket(zmq.PUSH)
		assertError(err)
		defer sender.Close()
		err = sender.Connect(*outputEndpoint)
		assertError(err)
		for {
			select {
			case ev := <-watcher.Event:
				log.Println("Event:", ev)
				if ev.IsCreate() {
					if isDir(ev.Name) {
						err = filepath.Walk(ev.Name, func(path string, info os.FileInfo, err error) error {
							if err != nil {
								return err
							}
							if info.IsDir() {
								watcher.Watch(path)
								log.Println("Added to watch:", path)
							}
							return nil
						})
						if err != nil {
							log.Println("Error walking directory:", err.Error())
						}
					} else {
						sender.SendMultipart(runtime.NewPacket([]byte(ev.Name)), 0)
					}
				} else if ev.IsDelete() {
					watcher.RemoveWatch(ev.Name)
					log.Println("Removed from watch:", ev.Name)
				}
			case err := <-watcher.Error:
				log.Println("Error:", err)
			}
		}
	}()

	// Main loop
	log.Println("Started")
	for {
		ip, err := receiver.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			continue
		}

		dir := string(ip[1])
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				watcher.Watch(path)
				log.Println("Added to watch:", path)
			}
			return nil
		})
		if err != nil {
			log.Printf("ERROR openning file %s: %s", dir, err.Error())
			if errorSocket != nil {
				errorSocket.SendMultipart(runtime.NewPacket([]byte(err.Error())), zmq.NOBLOCK)
			}
			continue
		}
	}
}
