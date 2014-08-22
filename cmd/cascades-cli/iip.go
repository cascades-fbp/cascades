package main

import (
	"cascades/log"
	"cascades/runtime"
	zmq "github.com/alecthomas/gozmq"
	"github.com/spf13/cobra"
	"os"
	"time"
)

//
// Run command
//
var cmdIIP = &cobra.Command{
	Use:   "iip",
	Short: "Manually send IIP to a port",
	Long: `Manually send IIP to a port. The IIP content is defined via content flag.
			The target port is defined by endpoint flag in the format of ZMQ's socket.`,
	Run: cmdIIPFunc,
}

//
// Run command handler function
//
func cmdIIPFunc(cmd *cobra.Command, args []string) {
	if endpoint == "" {
		cmd.Usage()
		return
	}

	context, _ := zmq.NewContext()
	defer context.Close()

	// Socket to send messages On
	sender, _ := context.NewSocket(zmq.PUSH)
	defer sender.Close()
	sender.Connect(endpoint)

	log.SystemOutput("Sending IIP to a given port...")
	sender.SendMultipart(runtime.NewPacket([]byte(content)), 0)
	cmd.Println("Done!")

	log.SystemOutput("Give 0MQ time to deliver before stopping...")
	time.Sleep(1e9)
	log.SystemOutput("Stopped")
	os.Exit(0)
}
