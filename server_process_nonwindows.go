//+build !windows

package servermanager

import (
	"os"
	"os/exec"
	"syscall"
)

const serverExecutablePath = "acServer"

func kill(process *os.Process) error {
	pgid, err := syscall.Getpgid(process.Pid)

	if err != nil {
		return err
	}

	return syscall.Kill(-pgid, syscall.SIGINT|syscall.SIGKILL)
}

func buildCommand(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd
}
