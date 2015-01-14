package runtime

import (
	"os"
	"os/exec"
	"syscall"
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
// Signal sends signal to a process
//
func (p *Process) Signal(signal syscall.Signal) {
	if p.Running() {
		group, _ := os.FindProcess(-1 * p.Pid())
		group.Signal(signal)
	}
}
