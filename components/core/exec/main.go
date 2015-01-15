package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"

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
	cmdCh, outCh, errCh       chan bool
	err                       error
)

func validateArgs() {
	if *cmdEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func openPorts() {
	cmdPort, err = utils.CreateInputPort("exec.cmd", *cmdEndpoint, cmdCh)
	utils.AssertError(err)

	if *outputEndpoint != "" {
		outPort, err = utils.CreateOutputPort("exec.out", *outputEndpoint, outCh)
		utils.AssertError(err)
	}

	if *errorEndpoint != "" {
		errPort, err = utils.CreateOutputPort("exec.err", *errorEndpoint, errCh)
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
	cmdCh = make(chan bool)
	outCh = make(chan bool)
	errCh = make(chan bool)
	go func() {
		select {
		case <-cmdCh:
			log.Println("CMD port is closed. Interrupting execution")
			ch <- syscall.SIGTERM
		case <-outCh:
			log.Println("OUT port is closed. Interrupting execution")
			ch <- syscall.SIGTERM
		case <-errCh:
			log.Println("ERR port is closed. Interrupting execution")
			ch <- syscall.SIGTERM
		}
	}()

	openPorts()
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
