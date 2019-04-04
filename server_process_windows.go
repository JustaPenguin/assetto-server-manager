//+build windows

package servermanager

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
)

const serverExecutablePath = "acServer.exe"

func kill(process *os.Process) error {
	err := process.Kill()

	if err != nil {
		logrus.WithError(err).Errorf("Initial attempt at killing windows process (process.Kill) failed")
		return exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", process.Pid)).Run()
	}

	return nil
}

func buildCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, command, args...)

	return cmd
}
