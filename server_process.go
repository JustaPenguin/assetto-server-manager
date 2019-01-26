package servermanager

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"sync"
)

const serverExecutablePath = "acServer"

var ErrServerAlreadyRunning = errors.New("servermanager: assetto corsa server is already running")

// AssettoServerProcess manages the assetto corsa server process.
type AssettoServerProcess struct {
	cmd *exec.Cmd

	out   *bytes.Buffer
	mutex sync.Mutex
}

// Logs outputs the server logs
func (as *AssettoServerProcess) Logs() string {
	return as.out.String()
}

// Start the assetto server. If it's already running, an ErrServerAlreadyRunning is returned.
func (as *AssettoServerProcess) Start() error {
	if as.Status() {
		return ErrServerAlreadyRunning
	}

	as.mutex.Lock()
	defer as.mutex.Unlock()

	as.cmd = exec.Command(filepath.Join(ServerInstallPath, serverExecutablePath))
	as.cmd.Dir = ServerInstallPath

	if as.out == nil {
		as.out = new(bytes.Buffer)
	}

	as.cmd.Stdout = as.out
	as.cmd.Stderr = as.out

	err := as.cmd.Start()

	if err != nil {
		as.cmd = nil
		return err
	}

	return nil
}

// Restart the assetto server.
func (as *AssettoServerProcess) Restart() error {
	if as.Status() {
		err := as.Stop()

		if err != nil {
			return err
		}
	}

	return as.Start()
}

// Status of the server. returns true if running
func (as *AssettoServerProcess) Status() bool {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	return as.cmd != nil && as.cmd.Process != nil
}

// Stop the assetto server.
func (as *AssettoServerProcess) Stop() error {
	if !as.Status() {
		return nil
	}

	as.mutex.Lock()
	defer as.mutex.Unlock()

	err := as.cmd.Process.Kill()

	if err != nil {
		return err
	}

	as.cmd = nil

	return nil
}
