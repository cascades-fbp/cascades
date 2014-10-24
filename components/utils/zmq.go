package utils

import (
	"fmt"
	"os"

	zmq "github.com/pebbe/zmq4"
)

// Print the error message if err is not nil & exist with status code 1
func AssertError(err error) {
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}
}

// Create a ZMQ PULL socket & bind to a given endpoint
func CreateInputPort(endpoint string) (socket *zmq.Socket, err error) {
	socket, err = zmq.NewSocket(zmq.PULL)
	if err != nil {
		return nil, err
	}
	err = socket.Bind(endpoint)
	if err != nil {
		return nil, err
	}
	return socket, nil
}

// Create a ZMQ PUSH socket & connect to a given endpoint
func CreateOutputPort(endpoint string) (socket *zmq.Socket, err error) {
	socket, err = zmq.NewSocket(zmq.PUSH)
	if err != nil {
		return nil, err
	}
	err = socket.Connect(endpoint)
	if err != nil {
		return nil, err
	}
	return socket, nil
}
