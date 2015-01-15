package main

import (
	"bufio"
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
	fileEndpoint   = flag.String("port.file", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	errorEndpoint  = flag.String("port.err", "", "Component's error port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	filePort, outPort, errPort *zmq.Socket
	fileCh, outCh, errCh       chan bool
	err                        error
)

func validateArgs() {
	if *fileEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *outputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func openPorts() {
	filePort, err = utils.CreateInputPort("readfile.file", *fileEndpoint, fileCh)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort("readfile.out", *outputEndpoint, outCh)
	utils.AssertError(err)

	if *errorEndpoint != "" {
		errPort, err = utils.CreateOutputPort("readfile.err", *errorEndpoint, errCh)
		utils.AssertError(err)
	}
}

func closePorts() {
	filePort.Close()
	outPort.Close()
	if errPort != nil {
		errPort.Close()
	}
	zmq.Term()
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

	ch := utils.HandleInterruption()
	fileCh = make(chan bool)
	outCh = make(chan bool)
	errCh = make(chan bool)

	openPorts()
	defer closePorts()

	ports := 1
	if outPort != nil {
		ports++
	}
	if errPort != nil {
		ports++
	}

	waitCh := make(chan bool)
	fileExitCh := make(chan bool, 1)
	go func(num int) {
		total := 0
		for {
			select {
			case v := <-fileCh:
				if v {
					total++
				} else {
					fileExitCh <- true
				}
			case v := <-outCh:
				if !v {
					log.Println("OUT port is closed. Interrupting execution")
					ch <- syscall.SIGTERM
				} else {
					total++
				}
			case v := <-errCh:
				if !v {
					log.Println("ERR port is closed. Interrupting execution")
					ch <- syscall.SIGTERM
				} else {
					total++
				}
			}
			if total >= num && waitCh != nil {
				waitCh <- true
			}
		}
	}(ports)

	log.Println("Waiting for port connections to establish... ")
	select {
	case <-waitCh:
		log.Println("Ports connected")
		waitCh = nil
	case <-time.Tick(30 * time.Second):
		log.Println("Timeout: port connections were not established within provided interval")
		os.Exit(1)
	}

	log.Println("Started...")
	for {
		ip, err := filePort.RecvMessageBytes(zmq.DONTWAIT)
		if err != nil {
			select {
			case <-fileExitCh:
				log.Println("FILE port is closed. Interrupting execution")
				ch <- syscall.SIGTERM
				break
			default:
				// IN port is still open
			}
			time.Sleep(2 * time.Second)
			continue
		}
		if !runtime.IsValidIP(ip) {
			continue
		}

		filepath := string(ip[1])
		f, err := os.Open(filepath)
		if err != nil {
			log.Printf("ERROR openning file %s: %s", filepath, err.Error())
			if errPort != nil {
				errPort.SendMessage(runtime.NewPacket([]byte(err.Error())))
			}
			continue
		}

		outPort.SendMessage(runtime.NewOpenBracket())
		outPort.SendMessage(ip)

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			outPort.SendMessage(runtime.NewPacket(scanner.Bytes()))
		}
		if err = scanner.Err(); err != nil && errPort != nil {
			errPort.SendMessage(runtime.NewPacket([]byte(err.Error())))
		}
		f.Close()

		outPort.SendMessage(runtime.NewCloseBracket())
	}
}
