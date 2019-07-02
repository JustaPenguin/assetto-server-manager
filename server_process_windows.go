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
	//apparently running Process.Kill() and THEN Taskill is a no-go
	//using only taskkill /t /f works instead, so it's pointless to use Process.Kill() at all - Alberto
	exec.Command("TASKKILL", "/T", "/F", "/PID", fmt.Sprintf("%d", process.Pid)).Run()
	return nil
}

func buildCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, command, args...)

	return cmd
}
