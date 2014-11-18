package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/template"

	"github.com/cascades-fbp/cascades/components/utils"
	"github.com/cascades-fbp/cascades/runtime"
	zmq "github.com/cascades-fbp/cascades/Godeps/_workspace/src/github.com/pebbe/zmq4"
)

var (
	// Flags
	tplEndpoint    = flag.String("port.tpl", "", "Component's options port endpoint")
	inputEndpoint  = flag.String("port.in", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")

	// Internal
	tplPort, inPort, outPort *zmq.Socket
	err                      error
)

func validateArgs() {
	if *tplEndpoint == "" {
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
	tplPort, err = utils.CreateInputPort(*tplEndpoint)
	utils.AssertError(err)

	inPort, err = utils.CreateInputPort(*inputEndpoint)
	utils.AssertError(err)

	outPort, err = utils.CreateOutputPort(*outputEndpoint)
	utils.AssertError(err)
}

func closePorts() {
	tplPort.Close()
	inPort.Close()
	outPort.Close()
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

	openPorts()
	defer closePorts()

	ch := utils.HandleInterruption()
	err = runtime.SetupShutdownByDisconnect(inPort, "template.in", ch)
	utils.AssertError(err)

	log.Println("Waiting for template...")
	var (
		t  *template.Template
		ip [][]byte
	)
	for {
		ip, err = tplPort.RecvMessageBytes(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			log.Println("Invalid IP:", ip)
			continue
		}
		t = template.New("Current template")
		t, err = t.Parse(string(ip[1]))
		if err != nil {
			log.Println("Failed to configure component:", err.Error())
			continue
		}
		break
	}

	log.Println("Started...")
	var (
		buf  *bytes.Buffer
		data map[string]interface{}
	)
	for {
		ip, err := inPort.RecvMessageBytes(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			continue
		}

		err = json.Unmarshal(ip[1], &data)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		buf = bytes.NewBufferString("")
		err = t.Execute(buf, data)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		outPort.SendMessage(runtime.NewPacket(buf.Bytes()))
	}
}
