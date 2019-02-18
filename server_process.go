package servermanager

import (
	"bytes"
	"errors"
	"net"
	"os/exec"
	"path/filepath"
	"sync"
)

type ServerProcess interface {
	Logs() string
	Start() error
	Stop() error
	Restart() error
	IsRunning() bool
}

var AssettoProcess ServerProcess

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
	if as.IsRunning() {
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
	if as.IsRunning() {
		err := as.Stop()

		if err != nil {
			return err
		}
	}

	return as.Start()
}

// IsRunning of the server. returns true if running
func (as *AssettoServerProcess) IsRunning() bool {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	return as.cmd != nil && as.cmd.Process != nil
}

// Stop the assetto server.
func (as *AssettoServerProcess) Stop() error {
	if !as.IsRunning() {
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

func FreeUDPPort() (int, error) {
	addr, err := net.ResolveUDPAddr("udp", "localhost:0")

	if err != nil {
		return 0, err
	}

	l, err := net.ListenUDP("udp", addr)

	if err != nil {
		return 0, err
	}

	defer l.Close()

	return l.LocalAddr().(*net.UDPAddr).Port, nil
}
