package main

import (
	"bytes"
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
	// flags
	cmdEndpoint    = flag.String("port.cmd", "", "Component's options port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	errorEndpoint  = flag.String("port.err", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	cmdPort, outPort, errPort *zmq.Socket
	err                       error
)

func validateArgs() {
	if *cmdEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func openPorts(termCh chan os.Signal) {
	cmdPort, err = utils.CreateMonitoredInputPort("exec.cmd", *cmdEndpoint, termCh)
	utils.AssertError(err)

	if *outputEndpoint != "" {
		outPort, err = utils.CreateOutputPort(*outputEndpoint)
		utils.AssertError(err)
	}

	if *errorEndpoint != "" {
		errPort, err = utils.CreateOutputPort(*errorEndpoint)
		utils.AssertError(err)
	}
}

func closePorts() {
	cmdPort.Close()
	if outPort != nil {
		outPort.Close()
	}
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
		ip, err := cmdPort.RecvMessageBytes(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			continue
		}
		out, err := executeCommand(string(ip[1]))
		if err != nil {
			log.Println(err.Error())
			if errPort != nil {
				errPort.SendMessage(runtime.NewPacket([]byte(err.Error())))
			}
			continue
		}
		out = bytes.Replace(out, []byte("\n"), []byte(""), -1)
		log.Println(string(out))
		if outPort != nil {
			outPort.SendMessage(runtime.NewPacket(out))
		}
	}
}
