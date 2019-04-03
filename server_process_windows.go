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
	err := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", process.Pid)).Run()

	if err != nil {
		logrus.WithError(err).Errorf("Initial attempt at killing windows process (taskkill) failed")
		return process.Kill()
	}

	return nil
}

func buildCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	args = append([]string{"/c", "start", "/wait", command}, args...)

	cmd := exec.CommandContext(ctx, "cmd", args...)

	return cmd
}
