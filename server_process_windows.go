//+build windows

package servermanager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

const serverExecutablePath = "acServer.exe"

func kill(process *os.Process) error {
	return exec.Command("TASKKILL", "/T", "/F", "/PID", fmt.Sprintf("%d", process.Pid)).Run()
}

func buildCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, command, args...)

	return cmd
}
