//+build windows

package servermanager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

const ServerExecutablePath = "acServer.exe"

func getProcess(cmd *exec.Cmd) *os.Process {
	return cmd.Process
}

func terminate(ps *os.Process) error {
	// Windows does not support SIGINT or SIGTERM, just nuke it..
	return kill(ps)
}

func kill(ps *os.Process) error {
	// Process.Kill() is unreliable for Windows.  Use less aesthetic but reliable method...
	return exec.Command("TASKKILL", "/T", "/F", "/PID", fmt.Sprintf("%d", ps.Pid)).Run()
}

func buildCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, command, args...)
}
