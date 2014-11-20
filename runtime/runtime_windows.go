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
	shutdownMutex.Lock()
	log.SystemOutput("Shutdown...")

	for name, ps := range self.processes {
		log.SystemOutput(fmt.Sprintf("terminating %s", name))
		ps.cmd.Process.Signal(os.Kill)
	}

	shutdownMutex.Unlock()
	os.Exit(1)
}
