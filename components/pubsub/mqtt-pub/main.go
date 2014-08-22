package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	zmq "github.com/alecthomas/gozmq"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	optionsEndpoint = flag.String("port.options", "", "Options endpoint to configure MQTT connection")
	inputEndpoint   = flag.String("port.in", "", "Component's input port endpoint")
)

func main() {
	flag.Parse()
	if *optionsEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *inputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.SetFlags(0)

	var (
		err    error
		qos    int
		topic  string
		client *mqtt.MqttClient
	)

	context, _ := zmq.NewContext()
	defer context.Close()

	optSocket, _ := context.NewSocket(zmq.PULL)
	defer optSocket.Close()
	optSocket.Bind(*optionsEndpoint)

	rcvSocket, _ := context.NewSocket(zmq.PULL)
	defer rcvSocket.Close()
	rcvSocket.Bind(*inputEndpoint)

	defer safeDisconnect(client)

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

	log.Println("Started")

	pollItems := zmq.PollItems{
		zmq.PollItem{Socket: optSocket, Events: zmq.POLLIN},
		zmq.PollItem{Socket: rcvSocket, Events: zmq.POLLIN},
	}

	for {
		_, err = zmq.Poll(pollItems, -1)
		if err != nil {
			log.Println("Error polling ports:", err.Error())
			continue
		}
		switch {
		case pollItems[0].REvents&zmq.POLLIN != 0:
			data, err := optSocket.Recv(0)
			if err != nil {
				log.Printf("Failed to receive options. Error: %s", err.Error())
				continue
			}
			broker, clientId, t, cleanSession, q, err := parseOptionsUri(string(data))
			if err != nil {
				log.Printf("Failed to parse connection uri. Error: %s", err.Error())
				continue
			}
			client, err = createMqttClient(broker, clientId, cleanSession)
			if err != nil {
				log.Printf("Failed to create MQTT client. Error: %s", err.Error())
				continue
			}
			optSocket.Close()
			qos = q
			topic = t
			publishData(rcvSocket, client, qos, topic)
		case pollItems[1].REvents&zmq.POLLIN != 0:
			if client == nil {
				// No client, don't consume this IP and wait for options
				continue
			}
			publishData(rcvSocket, client, qos, topic)
		}
	}
}

//
// Publish data if available
//
func publishData(input *zmq.Socket, client *mqtt.MqttClient, qos int, topic string) {
	if client == nil {
		return
	}
	msg, _ := input.RecvMultipart(0)

	if len(msg) == 3 {
		// 'topic qos message'
		buf := bytes.NewBuffer(msg[1])
		if qint, err := binary.ReadVarint(buf); err != nil {
			_ = client.Publish(mqtt.QoS(qint), string(msg[0]), msg[2])
			log.Printf("Published to MQTT: topic %s qos: %d msg: %s\n",
				string(msg[0]), qint, string(msg[0]))
		} else {
			log.Printf("Error parsing QoS %x", msg[1])
		}

	} else if len(msg) == 1 {
		// 'message' only
		if topic != "" {
			_ = client.Publish(mqtt.QoS(qos), topic, msg[0])
			log.Printf("Published to MQTT: topic %s qos: %d msg: %s\n",
				topic, qos, string(msg[0]))
		} else {
			log.Printf("Cannot publish a message %s: topic is not set\n", string(msg[0]))
		}

	} else {
		//unknown
		log.Println("Unknown IP received:", string(msg[0]))
	}
}

//
// Create a default client id based on the hostname and current time (nanoseconds)
//
func defaultClientId() string {
	hn, _ := os.Hostname()
	return "mqtt-pub_" + strings.Split(hn, ".")[0] + strconv.Itoa(time.Now().Nanosecond())
}

//
// Parses URI in the following format:
// tcp://127.0.0.1:1883/topic?clientId=...&clean=true&qos=0
// Keep in mind # should be expressed as %23
//
func parseOptionsUri(uri string) (broker, clientId, topic string, cleanSession bool, qos int, err error) {
	url, err := url.Parse(uri)
	if err != nil {
		return
	}

	broker = fmt.Sprintf("%s://%s", url.Scheme, url.Host)

	clientId = url.Query().Get("clientId")
	if clientId == "" {
		clientId = defaultClientId()
	}

	topic = url.Path

	cleanSession = true
	if url.Query().Get("clean") != "" && url.Query().Get("clean") != "true" {
		cleanSession = false
	}

	qos, err = strconv.Atoi(url.Query().Get("qos"))

	return
}

//
// Creates MQTT client
//
func createMqttClient(broker, clientId string, cleanSession bool) (*mqtt.MqttClient, error) {
	opts := mqtt.NewClientOptions().
		SetBroker(broker).
		SetClientId(clientId).
		SetCleanSession(cleanSession).
		SetTraceLevel(mqtt.Off)

	c := mqtt.NewClient(opts)
	if _, err := c.Start(); err != nil {
		return nil, err
	}

	return c, nil
}

//
// Cleanup
//
func safeDisconnect(client *mqtt.MqttClient) {
	log.Println("OK!")
	if client != nil {
		client.Disconnect(1e6)
	}
}
