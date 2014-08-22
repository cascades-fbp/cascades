package main

import (
	"flag"
	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	zmq "github.com/alecthomas/gozmq"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	outputEndpoint = flag.String("port.out", "", "Component's output port endpoint")
	broker         = flag.String("meta.broker", "tcp://127.0.0.1:1883", "MQTT broker address")
	clientid       = flag.String("meta.clientid", "", "A client ID for the connection")
	topic          = flag.String("meta.topic", "/", "Topic for messages published in MQTT")
	qos            = flag.Int("meta.qos", 0, "QoS for messages published in MQTT")
)

var sndSocket *zmq.Socket

// will be run as a go routine by mqtt.StartSubscription
func messageReceived(client *mqtt.MqttClient, msg mqtt.Message) {
	log.Printf("Received: %s %d %s\n", msg.Topic(), msg.QoS(), string(msg.Payload()))

	// topic
	if err := sndSocket.Send([]byte(msg.Topic()), zmq.SNDMORE); err != nil {
		log.Fatal("Error sending:", err)
	}
	// qos
	if err := sndSocket.Send([]byte{byte(msg.QoS())}, zmq.SNDMORE); err != nil {
		log.Fatal("Error sending:", err)
	}
	// payload
	if err := sndSocket.Send(msg.Payload(), zmq.NOBLOCK); err != nil {
		log.Fatal("Error sending:", err)
	}
}

func main() {
	flag.Parse()
	if *outputEndpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *clientid == "" {
		hn, _ := os.Hostname()
		cid := "mqtt-sub_" + strings.Split(hn, ".")[0] + strconv.Itoa(time.Now().Nanosecond())
		clientid = &cid
	}

	log.SetFlags(0)

	context, _ := zmq.NewContext()
	defer context.Close()

	// Socket to write messages into
	sndSocket, _ = context.NewSocket(zmq.PUSH)
	defer sndSocket.Close()
	err := sndSocket.Connect(*outputEndpoint)

	// MQTT client initialization
	opts := mqtt.NewClientOptions().
		SetBroker(*broker).
		SetClientId(*clientid).
		SetCleanSession(true).
		SetTraceLevel(mqtt.Off)

	c := mqtt.NewClient(opts)
	_, err = c.Start()
	if err != nil {
		panic(err)
	}

	log.Println("Started")

	// Ctrl+C handling
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range ch {
			// close mqtt connection
			log.Println("Disconnecting from MQTT broker")
			c.Disconnect(0)

			// close 0mq socket
			time.Sleep(1e9)
			sndSocket.Close()

			log.Println("Stopped")
			os.Exit(0)
		}
	}()

	f, err := mqtt.NewTopicFilter(*topic, byte(*qos))
	if err != nil {
		panic(err)
	}
	c.StartSubscription(messageReceived, f)

	// Loop
	ticker := time.Tick(1 * time.Second)
	for _ = range ticker {
	}

}
