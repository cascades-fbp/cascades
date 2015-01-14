package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

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

func openPorts(termCh chan os.Signal) {
	filePort, err = utils.CreateMonitoredInputPort("readfile.file", *fileEndpoint, termCh)
	utils.AssertError(err)

	outPort, err = utils.CreateMonitoredOutputPort("readfile.out", *outputEndpoint, termCh)
	utils.AssertError(err)

	if *errorEndpoint != "" {
		errPort, err = utils.CreateOutputPort(*errorEndpoint)
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
	openPorts(ch)
	defer closePorts()

	log.Println("Started...")
	for {
		ip, err := filePort.RecvMessageBytes(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
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
