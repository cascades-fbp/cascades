// WARNING: This won't build on Windows and Plan9.

//
//  Handling Ctrl-C cleanly in C.
//

package main

import (
	zmq "github.com/cascades-fbp/cascades/Godeps/_workspace/src/github.com/pebbe/zmq4"

	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	//  Socket to talk to server
	fmt.Println("Connecting to hello world server...")
	client, _ := zmq.NewSocket(zmq.REQ)
	defer client.Close()
	client.Connect("tcp://localhost:5555")

	// Without signal handling, Go will exit on signal, even if the signal was caught by ZeroMQ
	chSignal := make(chan os.Signal, 1)
	signal.Notify(chSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

LOOP:
	for {
		client.Send("HELLO", 0)
		reply, err := client.Recv(0)
		if err != nil {
			if zmq.AsErrno(err) == zmq.Errno(syscall.EINTR) {
				// signal was caught by 0MQ
				log.Println("Client Recv:", err)
				break
			} else {
				// some error occured
				log.Panicln(err)
			}
		}

		fmt.Println("Client:", reply)
		time.Sleep(time.Second)

		select {
		case sig := <-chSignal:
			// signal was caught by Go
			log.Println("Signal:", sig)
			break LOOP
		default:
		}
	}
}
