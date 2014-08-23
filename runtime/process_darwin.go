package runtime

import (
	"fmt"
	"github.com/cascades-fbp/cascades/log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

//
// Start a process
//
func (p *Process) Start() {
	command := []string{"/bin/bash", p.shellArgument(), p.Command()}
	p.cmd = exec.Command(command[0], command[1:]...)
	p.cmd.Dir = p.Root
	p.cmd.Env = p.envAsArray()
	p.cmd.Stdin = p.Stdin
	p.cmd.Stdout = p.Stdout
	p.cmd.Stderr = p.Stderr
	if !p.Interactive {
		p.cmd.SysProcAttr = &syscall.SysProcAttr{}
		p.cmd.SysProcAttr.Setsid = true
	}
	p.cmd.Start()
}

//
// Send signal to a process
//
func (p *Process) Signal(signal syscall.Signal) {
	if p.Running() {
		group, _ := os.FindProcess(-1 * p.Pid())
		group.Signal(signal)
	}
}

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
