package runtime

import (
	"fmt"
	"github.com/cascades-fbp/cascades/log"
	"syscall"
	"time"
)

//
// Shutdown the network
//
func (self *Runtime) Shutdown() {
	log.SystemOutput("Shutdown...")

	shutdownMutex.Lock()
	for name, ps := range self.processes {
		log.SystemOutput(fmt.Sprintf("sending SIGTERM to %s", name))
		ps.Signal(syscall.SIGTERM)
	}

	if len(self.processes) == 0 {
		self.Done <- true
	}

	go func() {
		time.Sleep(3 * time.Second)
		for name, ps := range self.processes {
			log.SystemOutput(fmt.Sprintf("sending SIGKILL to %s", name))
			ps.Signal(syscall.SIGKILL)
		}
		self.Done <- true
	}()

	shutdownMutex.Unlock()
}
