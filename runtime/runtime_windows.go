package runtime

import (
	"fmt"
	"os"

	"github.com/cascades-fbp/cascades/log"
)

//
// Shutdown the network
//
func (self *Runtime) Shutdown() {
	shutdown_mutex.Lock()
	log.SystemOutput("Shutdown...")

	for name, ps := range self.processes {
		log.SystemOutput(fmt.Sprintf("terminating %s", name))
		ps.cmd.Process.Signal(os.Kill)
	}
	os.Exit(1)
}
