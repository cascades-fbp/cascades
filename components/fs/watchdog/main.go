package main

import (
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	"github.com/howeyc/fsnotify"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var (
	// Flafs
	inputEndpoint  = flag.String("port.dir", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.created", "", "Component's output port endpoint")
	errorEndpoint  = flag.String("port.err", "", "Component's error port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	context                  *zmq.Context
	inPort, outPort, errPort *zmq.Socket
	ip                       [][]byte
	err                      error
)

func validateArgs() {
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *outputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func openPorts() {
	context, err = zmq.NewContext()
	utils.AssertError(err)

	inPort, err = utils.CreateInputPort(context, *inputEndpoint)
	utils.AssertError(err)

	if *errorEndpoint != "" {
		errPort, err = utils.CreateOutputPort(context, *errorEndpoint)
		utils.AssertError(err)
	}
}

func closePorts() {
	inPort.Close()
	if outPort != nil {
		outPort.Close()
	}
	if errPort != nil {
		errPort.Close()
	}
	context.Close()
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

	if *jsonFlag {
		doc, _ := registryEntry.JSON()
		fmt.Println(string(doc))
		os.Exit(0)
	}

	log.SetFlags(0)
	if *debug {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	validateArgs()

	openPorts()
	defer closePorts()

	utils.HandleInterruption()

	// Setup watcher
	watcher, err := fsnotify.NewWatcher()
	utils.AssertError(err)
	defer watcher.Close()

	// Process events
	go func() {
		//  Socket to send messages to task sink
		outPort, err = utils.CreateOutputPort(context, *outputEndpoint)
		utils.AssertError(err)
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
								// we need to watch every subdirectory explicitely
								watcher.Watch(path)
								log.Println("Added to watch:", path)
							} else {
								// Consider every file found in the created directory as just created
								outPort.SendMultipart(runtime.NewPacket([]byte(path)), 0)
							}
							return nil
						})
						if err != nil {
							log.Println("Error walking directory:", err.Error())
						}
					} else {
						outPort.SendMultipart(runtime.NewPacket([]byte(ev.Name)), 0)
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
		ip, err := inPort.RecvMultipart(0)
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
			if errPort != nil {
				errPort.SendMultipart(runtime.NewPacket([]byte(err.Error())), 0)
			}
			continue
		}
	}
}
