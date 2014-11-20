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
	command := []string{"cmd", "/C", p.Command()}
	p.cmd = exec.Command(command[0], command[1:]...)
	p.cmd.Dir = p.Root
	p.cmd.Env = p.envAsArray()
	p.cmd.Stdin = p.Stdin
	p.cmd.Stdout = p.Stdout
	p.cmd.Stderr = p.Stderr
	p.cmd.Start()
}

//
// Send signal to a process
//
func (p *Process) Signal(signal syscall.Signal) {
	group, _ := os.FindProcess(-1 * p.cmd.Process.Pid)
	group.Signal(signal)
}
