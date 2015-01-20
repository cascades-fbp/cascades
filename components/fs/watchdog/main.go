package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	"github.com/howeyc/fsnotify"
	zmq "github.com/pebbe/zmq4"
)

var (
	// Flafs
	inputEndpoint  = flag.String("port.dir", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.created", "", "Component's output port endpoint")
	errorEndpoint  = flag.String("port.err", "", "Component's error port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	inPort, outPort, errPort *zmq.Socket
	outCh, errCh             chan bool
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
	inPort, err = utils.CreateInputPort("fs/watchdog.in", *inputEndpoint, nil)
	utils.AssertError(err)

	if *errorEndpoint != "" {
		errPort, err = utils.CreateOutputPort("fs/watchdog.err", *errorEndpoint, errCh)
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
	zmq.Term()
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

	// Setup watcher
	watcher, err := fsnotify.NewWatcher()
	utils.AssertError(err)
	defer watcher.Close()

	ch := utils.HandleInterruption()
	outCh = make(chan bool)
	errCh = make(chan bool)

	openPorts()
	defer closePorts()

	// Process events
	go func() {
		//  Socket to send messages to task sink
		outPort, err = utils.CreateOutputPort("fs/watchdog.out", *outputEndpoint, errCh)
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
								outPort.SendMessage(runtime.NewPacket([]byte(path)))
							}
							return nil
						})
						if err != nil {
							log.Println("Error walking directory:", err.Error())
						}
					} else {
						outPort.SendMessage(runtime.NewPacket([]byte(ev.Name)))
					}
				} else if ev.IsDelete() && isDir(ev.Name) {
					watcher.RemoveWatch(ev.Name)
					log.Println("Removed from watch:", ev.Name)
				}
			case err := <-watcher.Error:
				log.Println("Error:", err)
			}
		}
	}()

	go func() {
		for {
			select {
			case v := <-outCh:
				if !v {
					log.Println("CREATED port is closed. Interrupting execution")
					ch <- syscall.SIGTERM
					break
				}
			case v := <-errCh:
				if !v {
					log.Println("ERR port is closed. Interrupting execution")
					ch <- syscall.SIGTERM
					break
				}
			}
		}
	}()

	// Main loop
	log.Println("Started")
	for {
		ip, err := inPort.RecvMessageBytes(0)
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
			log.Printf("ERROR opening file %s: %s", dir, err.Error())
			if errPort != nil {
				errPort.SendMessage(runtime.NewPacket([]byte(err.Error())))
			}
			continue
		}
	}
}
