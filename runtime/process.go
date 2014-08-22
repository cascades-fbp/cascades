package runtime

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Map of key/values to pass as env variables to a process
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

// IIP model (sent when processes started)
type ProcessIIP struct {
	Payload string
	Socket  string
}

// Process constructor
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

func (p *Process) Running() bool {
	return (p.cmd.Process != nil)
}

func (p *Process) Pid() int {
	return p.cmd.Process.Pid
}

func (p *Process) Wait() {
	p.cmd.Wait()
}

func (p *Process) Command() string {
	return fmt.Sprintf("%s %s", p.Executable, p.Arguments())
}

func (p *Process) Arguments() string {
	args := ""
	for k, v := range p.Args {
		args = fmt.Sprintf("%s%s=\"%s\" ", args, k, v)
	}
	return strings.ToLower(args)
}

func (p *Process) shellArgument() string {
	if p.Interactive {
		return "-ic"
	} else {
		return "-c"
	}
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
