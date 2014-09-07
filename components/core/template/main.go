package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/cascades-fbp/cascades/runtime"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"
)

var (
	tplEndpoint    = flag.String("port.tpl", "", "Component's options port endpoint")
	inputEndpoint  = flag.String("port.in", "", "Component's input port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	jsonFlag       = flag.Bool("json", false, "Print component documentation in JSON")
	debug          = flag.Bool("debug", false, "Enable debug mode")
)

func assertError(err error) {
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}
}

func main() {
	flag.Parse()

	if *jsonFlag {
		doc, _ := registryEntry.JSON()
		fmt.Println(string(doc))
		os.Exit(0)
	}

	if *tplEndpoint == "" || *inputEndpoint == "" || *outputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.SetFlags(0)
	if *debug {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	var err error
	context, _ := zmq.NewContext()
	defer context.Close()

	//  Socket to receive messages on
	options, err := context.NewSocket(zmq.PULL)
	assertError(err)
	defer options.Close()
	err = options.Bind(*tplEndpoint)
	assertError(err)
	receiver, err := context.NewSocket(zmq.PULL)
	assertError(err)
	defer receiver.Close()
	err = receiver.Bind(*inputEndpoint)
	assertError(err)

	//  Socket to send messages to task sink
	sender, err := context.NewSocket(zmq.PUSH)
	assertError(err)
	defer sender.Close()
	err = sender.Connect(*outputEndpoint)
	assertError(err)

	// Ctrl+C handling
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range ch {
			log.Println("Give 0MQ time to deliver before stopping...")
			time.Sleep(1e9)
			log.Println("Stopped")
			os.Exit(0)
		}
	}()

	// Monitoring setup
	err = runtime.SetupShutdownByDisconnect(context, receiver, "template.in", ch)
	if err != nil {
		log.Println("Failed to setup monitoring:", err.Error())
		os.Exit(1)
	}

	// Wait for template
	ip, err := options.RecvMultipart(0)
	assertError(err)
	if !runtime.IsValidIP(ip) {
		err = fmt.Errorf("Received invalid IP at options port: %#v", ip)
	}
	assertError(err)
	t := template.New("current template")
	t, err = t.Parse(string(ip[1]))
	assertError(err)

	// Main loop
	var (
		buf  *bytes.Buffer
		data map[string]interface{}
	)
	log.Println("Started")
	for {
		ip, err := receiver.RecvMultipart(0)
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

		sender.SendMultipart(runtime.NewPacket(buf.Bytes()), zmq.NOBLOCK)
	}
}
