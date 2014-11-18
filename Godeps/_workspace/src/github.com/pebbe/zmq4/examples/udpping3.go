//
//  UDP ping command
//  Model 3, uses abstract network interface
//

package main

import (
	"github.com/cascades-fbp/cascades/Godeps/_workspace/src/github.com/pebbe/zmq4/examples/intface"

	"fmt"
	"log"
)

func main() {
	log.SetFlags(log.Lshortfile)
	iface := intface.New()
	for {
		msg, err := iface.Recv()
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%q\n", msg)
	}
}
