package main

import (
	"os"
	"os/exec"
)

func executeCommand(command string) ([]byte, error) {
	cmd := exec.Command("/bin/bash", "-c", command)
	cmd.Env = os.Environ()
	return cmd.Output()
}
