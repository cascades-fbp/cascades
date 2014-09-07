package main

import (
	"flag"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/cascades-fbp/cascades/runtime"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

var (
	cmdEndpoint    = flag.String("port.cmd", "", "Component's options port endpoint")
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	errorEndpoint  = flag.String("port.err", "", "Component's output port endpoint")
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

	if *cmdEndpoint == "" {
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
	cmdSock, err := context.NewSocket(zmq.PULL)
	assertError(err)
	defer cmdSock.Close()
	err = cmdSock.Bind(*cmdEndpoint)
	assertError(err)

	//  Socket to send messages to
	var outSock, errSock *zmq.Socket
	if *outputEndpoint != "" {
		outSock, err = context.NewSocket(zmq.PUSH)
		assertError(err)
		defer outSock.Close()
		err = outSock.Connect(*outputEndpoint)
		assertError(err)
	}
	if *errorEndpoint != "" {
		errSock, err = context.NewSocket(zmq.PUSH)
		assertError(err)
		defer errSock.Close()
		err = errSock.Connect(*errorEndpoint)
		assertError(err)
	}

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
	err = runtime.SetupShutdownByDisconnect(context, cmdSock, "exec.cmd", ch)
	if err != nil {
		log.Println("Failed to setup monitoring:", err.Error())
		os.Exit(1)
	}

	// Main loop
	log.Println("Started")
	for {
		ip, err := cmdSock.RecvMultipart(0)
		if err != nil {
			log.Println("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			continue
		}
		cmd := exec.Command("/bin/bash", "-c", string(ip[1]))
		cmd.Env = os.Environ()
		out, err := cmd.Output()
		if err != nil {
			log.Println(err.Error())
			if errSock != nil {
				errSock.SendMultipart(runtime.NewPacket([]byte(err.Error())), 0)
			}
			continue
		}
		log.Println(string(out))
		if outSock != nil {
			outSock.SendMultipart(runtime.NewPacket(out), 0)
		}
	}
}
