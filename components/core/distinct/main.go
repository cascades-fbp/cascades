package main

import (
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	zmq "github.com/pebbe/zmq4"
)

var (
	// Flags
	optionsEndpoint = flag.String("port.options", "", "Component's options port endpoint")
	inputEndpoint   = flag.String("port.in", "", "Component's input port endpoint")
	outputEndpoint  = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag        = flag.Bool("json", false, "Print component documentation in JSON")
	debug           = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	optionsPort, inPort, outPort *zmq.Socket
	inCh, outCh, loopCh          chan bool
	exitCh                       chan os.Signal
	opts                         *options
	localCache                   *Cache
	err                          error
)

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

	// Communication channels
	inCh = make(chan bool)
	outCh = make(chan bool)
	loopCh = make(chan bool)
	exitCh = make(chan os.Signal, 1)

	// Start the communication & processing logic
	go mainLoop()

	// Wait for the end...
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM)
	<-exitCh

	// Shutdown main loop with timeout (let it save the cache if required)
	select {
	case loopCh <- true:
		// let it save all stuff
		time.Sleep(2 * time.Second)
	case <-time.Tick(3 * time.Second):
	}

	log.Println("Done")
}

// mainLoop initiates all ports and handles the traffic
func mainLoop() {
	openPorts()
	defer closePorts()

	waitCh := make(chan bool)
	go func() {
		total := 0
		for {
			select {
			case v := <-inCh:
				if !v {
					log.Println("IN port is closed. Interrupting execution")
					exitCh <- syscall.SIGTERM
					break
				} else {
					total++
				}
			case v := <-outCh:
				if !v {
					log.Println("OUT port is closed. Interrupting execution")
					exitCh <- syscall.SIGTERM
					break
				} else {
					total++
				}
			}
			if total >= 2 && waitCh != nil {
				waitCh <- true
			}
		}
	}()

	log.Println("Waiting for port connections to establish... ")
	select {
	case <-waitCh:
		log.Println("Ports connected")
		waitCh = nil
	case <-time.Tick(30 * time.Second):
		log.Println("Timeout: port connections were not established within provided interval")
		exitCh <- syscall.SIGTERM
		return
	}

	log.Println("Waiting for options...")
	var (
		ip [][]byte
	)
	for {
		ip, err = optionsPort.RecvMessageBytes(0)
		if err != nil {
			continue
		}
		if !runtime.IsValidIP(ip) {
			log.Println("Invalid IP:", ip)
			continue
		}
		err = json.Unmarshal(ip[1], &opts)
		if err != nil {
			log.Println("Failed to resolve options:", err.Error())
			continue
		}
		log.Printf("Using options: %#v", opts)
		break
	}
	optionsPort.Close()

	/*
		if err = opts.Validate(); err != nil {
			log.Println("ERROR: Invalid options:", err.Error())
			ch <- syscall.SIGTERM
			return
		}
	*/

	localCache = NewCache(time.Duration(opts.DefaultExpiration)*time.Second, time.Duration(opts.CleanupInterval)*time.Second)
	if opts.IsPersistent() {
		log.Println("Cache is persistent. Using file", opts.File)
		err = localCache.LoadFile(opts.File)
		if err != nil {
			log.Println("WARNING: Failed to load cache from file", opts.File)
		}
	}

	log.Println("Started...")
loop:
	for {
		ip, err = inPort.RecvMessageBytes(zmq.DONTWAIT)
		if err != nil {
			select {
			case <-loopCh:
				log.Println("Main loop shutdown requested")
				break loop
			default:
			}
			time.Sleep(2 * time.Second)
			continue
		}

		if !runtime.IsValidIP(ip) {
			log.Println("Invalid IP:", ip)
			continue
		}

		key := fmt.Sprintf("%x", md5.Sum(ip[1]))
		if _, found := localCache.Get(key); found {
			log.Println("Cache HIT. Not forwarding this IP")
			continue
		}

		log.Println("Cache MISS. Forwarding")

		outPort.SendMessage(ip)

		localCache.Add(key, 1, 0)
	}

	if opts.IsPersistent() {
		log.Println("Saving current cache to", opts.File)
		err = localCache.SaveFile(opts.File)
		if err != nil {
			log.Println("ERROR saving cache to file:", err.Error())
		}
	}
}

// validateArgs checks all required flags
func validateArgs() {
	if *optionsEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *outputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

// openPorts create ZMQ sockets and start socket monitoring loops
func openPorts() {
	optionsPort, err = utils.CreateInputPort("distinct.options", *optionsEndpoint, nil)
	utils.AssertError(err)

	inPort, err = utils.CreateInputPort("distinct.in", *inputEndpoint, inCh)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort("distinct.out", *outputEndpoint, outCh)
	utils.AssertError(err)
}

// closePorts closes all active ports and terminates ZMQ context
func closePorts() {
	log.Println("Closing ports...")
	optionsPort.Close()
	inPort.Close()
	outPort.Close()
	zmq.Term()
}
