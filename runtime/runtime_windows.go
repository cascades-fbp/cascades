package runtime

import (
	"fmt"
	"os"

	"github.com/cascades-fbp/cascades/log"
)

//
// Shutdown the network
//
func ShutdownProcesses(of *OutletFactory) {
	shutdown_mutex.Lock()
	log.SystemOutput("Shutdown...")

	for name, ps := range processes {
		log.SystemOutput(fmt.Sprintf("terminating %s", name))
		ps.cmd.Process.Signal(os.Kill)
	}
	os.Exit(1)
}
