package main

import (
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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
	inCh, outCh                  chan bool
	opts                         *options
	localCache                   *Cache
	err                          error
)

// Options
type options struct {
	DefaultExpiration int    `json:"duration"`
	CleanupInterval   int    `json:"cleanup"`
	File              string `json:"file"`
}

func (o *options) IsPersistent() bool {
	return o.File != ""
}

func (o *options) Validate() error {
	if o.DefaultExpiration < 0 {
		o.DefaultExpiration = 0
	}
	if o.CleanupInterval < 0 {
		o.CleanupInterval = 0
	}
	if o.CleanupInterval < o.DefaultExpiration {
		o.CleanupInterval = o.DefaultExpiration + 10
	}
	if o.IsPersistent() {
		info, err := os.Stat(o.File)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("Received directory instead of a file: %s", o.File)
		}
	}
	return nil
}

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

func openPorts() {
	optionsPort, err = utils.CreateInputPort("distinct.options", *optionsEndpoint, nil)
	utils.AssertError(err)

	inPort, err = utils.CreateInputPort("distinct.in", *inputEndpoint, inCh)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort("distinct.out", *outputEndpoint, outCh)
	utils.AssertError(err)
}

func closePorts() {
	optionsPort.Close()
	inPort.Close()
	outPort.Close()
	zmq.Term()
}

// Main entry point
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

	ch := utils.HandleInterruption()
	inCh = make(chan bool)
	outCh = make(chan bool)

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
					ch <- syscall.SIGTERM
				} else {
					total++
				}
			case v := <-outCh:
				if !v {
					log.Println("OUT port is closed. Interrupting execution")
					ch <- syscall.SIGTERM
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
		os.Exit(1)
	}

	log.Println("Waiting for options...")
	var (
		ip [][]byte
	)
	for {
		ip, err = optionsPort.RecvMessageBytes(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			log.Println("Invalid IP:", ip)
			ch <- syscall.SIGTERM
			continue
		}
		err = json.Unmarshal(ip[1], &opts)
		if err != nil {
			log.Println("Failed to resolve options:", err.Error())
			ch <- syscall.SIGTERM
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
	defer func() {
		if opts.IsPersistent() {
			log.Println("Saving current cache to", opts.File)
			localCache.SaveFile(opts.File)
		}
	}()

	log.Println("Started...")
	for {
		ip, err = inPort.RecvMessageBytes(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
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

		localCache.Add(key, nil, 0)
	}
}
