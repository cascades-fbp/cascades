package main

import (
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var (
	// Flags
	inputEndpoint  = flag.String("port.dir", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.file", "", "Component's output port endpoint")
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

	outPort, err = utils.CreateOutputPort(context, *outputEndpoint)
	utils.AssertError(err)

	if *errorEndpoint != "" {
		errPort, err = utils.CreateOutputPort(context, *errorEndpoint)
		utils.AssertError(err)
	}
}

func closePorts() {
	inPort.Close()
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
	err = runtime.SetupShutdownByDisconnect(context, inPort, "fswalk.in", ch)
	utils.AssertError(err)

	log.Println("Started...")
	for {
		ip, err = inPort.RecvMultipart(0)
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
			if !info.IsDir() {
				outPort.SendMultipart(runtime.NewPacket([]byte(path)), 0)
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
