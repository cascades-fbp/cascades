package runtime

import (
	"fmt"
	"syscall"
	"time"

	"github.com/cascades-fbp/cascades/log"
)

//
// Shutdown the network
//
func (r *Runtime) Shutdown() {
	log.SystemOutput("Shutdown...")

	shutdownMutex.Lock()
	for name, ps := range r.processes {
		log.SystemOutput(fmt.Sprintf("sending SIGTERM to %s", name))
		ps.Signal(syscall.SIGTERM)
	}

	if len(r.processes) == 0 {
		r.Done <- true
	}

	go func() {
		time.Sleep(3 * time.Second)
		for name, ps := range r.processes {
			log.SystemOutput(fmt.Sprintf("sending SIGKILL to %s", name))
			ps.Signal(syscall.SIGKILL)
		}
		r.Done <- true
	}()

	shutdownMutex.Unlock()
}
