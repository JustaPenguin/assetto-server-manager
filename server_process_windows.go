//+build windows

package servermanager

import (
	"fmt"
	"os"
	"os/exec"
)

const serverExecutablePath = "acServer.exe"

func kill(process *os.Process) error {
	return exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(process.Pid)).Run()
}

func buildCommand(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)

	return cmd
}
