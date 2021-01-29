//+build !windows

package servermanager

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	"github.com/sirupsen/logrus"
)

const ServerExecutablePath = "acServer"

func getProcess(cmd *exec.Cmd) *os.Process {
	if pgid, err := syscall.Getpgid(cmd.Process.Pid); err != nil {
		logrus.WithError(err).Warnf("Failed to get process group for %d. Using pid instead.", pgid)
		return cmd.Process
	} else if ps, err := os.FindProcess(pgid); err != nil {
		logrus.WithError(err).Warnf("Failed to find process for %d. Using pid instead.", pgid)
		return cmd.Process
	} else {
		return ps
	}
}

func terminate(ps *os.Process) error {
	return syscall.Kill(-ps.Pid, syscall.SIGINT)
}

func kill(ps *os.Process) error {
	return syscall.Kill(-ps.Pid, syscall.SIGKILL)
}

func buildCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}
