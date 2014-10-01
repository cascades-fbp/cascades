package main

import (
	"bufio"
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	"io/ioutil"
	"log"
	"os"
)

var (
	// Flags
	fileEndpoint   = flag.String("port.file", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	errorEndpoint  = flag.String("port.err", "", "Component's error port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	context                    *zmq.Context
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

func openPorts() {
	context, err = zmq.NewContext()
	utils.AssertError(err)

	filePort, err = utils.CreateInputPort(context, *fileEndpoint)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort(context, *outputEndpoint)
	utils.AssertError(err)

	if *errorEndpoint != "" {
		errPort, err = utils.CreateOutputPort(context, *errorEndpoint)
		utils.AssertError(err)
	}
}

func closePorts() {
	filePort.Close()
	outPort.Close()
	if errPort != nil {
		errPort.Close()
	}
	context.Close()
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

	ch := utils.HandleInterruption()
	err = runtime.SetupShutdownByDisconnect(context, filePort, "readfile.file", ch)
	utils.AssertError(err)

	log.Println("Started...")
	for {
		ip, err := filePort.RecvMultipart(0)
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
				errPort.SendMultipart(runtime.NewPacket([]byte(err.Error())), 0)
			}
			continue
		}

		outPort.SendMultipart(runtime.NewOpenBracket(), 0)
		outPort.SendMultipart(ip, 0)

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			outPort.SendMultipart(runtime.NewPacket(scanner.Bytes()), 0)
		}
		if err = scanner.Err(); err != nil && errPort != nil {
			errPort.SendMultipart(runtime.NewPacket([]byte(err.Error())), 0)
		}
		f.Close()

		outPort.SendMultipart(runtime.NewCloseBracket(), 0)
	}
}
