package runtime

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Env is a map of key/values to pass as env variables to a process
type Env map[string]string

// Process model
type Process struct {
	Executable  string
	Args        map[string]string
	Env         Env
	Interactive bool
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Root        string

	cmd *exec.Cmd
}

// ProcessIIP is a model of IIP (sent when processes started)
type ProcessIIP struct {
	Payload string
	Socket  string
}

// NewProcess is a process constructor
func NewProcess(executable string) (p *Process) {
	p = new(Process)
	p.Executable = executable
	p.Args = make(map[string]string)
	p.Env = make(Env)
	p.Interactive = false
	p.Stdin = os.Stdin
	p.Stdout = os.Stdout
	p.Stderr = os.Stderr
	p.Root, _ = os.Getwd()
	return
}

// Running returns true is process is running
func (p *Process) Running() bool {
	return (p.cmd.Process != nil)
}

// Pid returns process' pid
func (p *Process) Pid() int {
	return p.cmd.Process.Pid
}

// Wait makes process command's wait
func (p *Process) Wait() {
	p.cmd.Wait()
}

// Command returns a process command to execute
func (p *Process) Command() string {
	return fmt.Sprintf("%s %s", p.Executable, p.Arguments())
}

// Arguments returns arguments string for a command
func (p *Process) Arguments() string {
	args := ""
	for k, v := range p.Args {
		if v == "" {
			args = fmt.Sprintf("%s%s ", args, k)
		} else {
			args = fmt.Sprintf("%s%s=\"%s\" ", args, k, v)
		}

	}
	return strings.ToLower(args)
}

func (p *Process) shellArgument() string {
	if p.Interactive {
		return "-ic"
	}
	return "-c"
}

func (p *Process) envAsArray() (env []string) {
	for _, pair := range os.Environ() {
		env = append(env, pair)
	}
	for name, val := range p.Env {
		env = append(env, fmt.Sprintf("%s=%s", name, val))
	}
	return
}
