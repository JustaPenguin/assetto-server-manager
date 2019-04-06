//+build !windows

package servermanager

import (
	"context"
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

	err = syscall.Kill(-pgid, syscall.SIGKILL)

	if err != nil {
		return err
	}

	err = syscall.Kill(-pgid, syscall.SIGINT)

	if err != nil {
		return err
	}

	return syscall.Kill(-process.Pid, syscall.SIGKILL|syscall.SIGINT)
}

func buildCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd
}
