package servermanager

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/mitchellh/go-ps"
	"github.com/sirupsen/logrus"
)

const MaxLogSizeBytes = 1e6

type ServerEventType string

const (
	EventTypeRace         ServerEventType = "RACE"
	EventTypeChampionship ServerEventType = "CHAMPIONSHIP"
)

type ServerProcess interface {
	Logs() string
	Start() error
	Stop() error
	Restart() error
	IsRunning() bool
	EventType() ServerEventType

	Done() <-chan struct{}
}

var AssettoProcess ServerProcess

// serverProcessHandler modifies the server process.
func serverProcessHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var txt string

	eventType := AssettoProcess.EventType()

	switch chi.URLParam(r, "action") {
	case "start":
		err = AssettoProcess.Start()
		txt = "started"
	case "stop":
		if eventType == EventTypeChampionship {
			err = championshipManager.StopActiveEvent()
		} else {
			err = AssettoProcess.Stop()
		}
		txt = "stopped"
	case "restart":
		if eventType == EventTypeChampionship {
			err = championshipManager.RestartActiveEvent()
		} else {
			err = AssettoProcess.Restart()
		}
		txt = "restarted"
	}

	noun := "Server"

	if eventType == EventTypeChampionship {
		noun = "Championship"
	}

	if err != nil {
		logrus.Errorf("could not change "+noun+" status, err: %s", err)
		AddErrFlashQuick(w, r, "Unable to change "+noun+" status")
	} else {
		AddFlashQuick(w, r, noun+" successfully "+txt)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

var ErrServerAlreadyRunning = errors.New("servermanager: assetto corsa server is already running")

// AssettoServerProcess manages the assetto corsa server process.
type AssettoServerProcess struct {
	cmd *exec.Cmd

	out   *logBuffer
	mutex sync.Mutex

	ctx context.Context
	cfn context.CancelFunc

	doneCh chan struct{}

	extraProcesses []*exec.Cmd
}

func NewAssettoServerProcess() *AssettoServerProcess {
	ctx, cfn := context.WithCancel(context.Background())

	return &AssettoServerProcess{
		ctx:    ctx,
		cfn:    cfn,
		doneCh: make(chan struct{}),
	}
}

func (as *AssettoServerProcess) Done() <-chan struct{} {
	return as.doneCh
}

// Logs outputs the server logs
func (as *AssettoServerProcess) Logs() string {
	if as.out == nil {
		return ""
	}

	return as.out.String()
}

// Start the assetto server. If it's already running, an ErrServerAlreadyRunning is returned.
func (as *AssettoServerProcess) Start() error {
	if as.IsRunning() {
		return ErrServerAlreadyRunning
	}

	as.mutex.Lock()
	defer as.mutex.Unlock()

	wd, err := os.Getwd()

	if err != nil {
		return err
	}

	var executablePath string

	if filepath.IsAbs(config.Steam.ExecutablePath) {
		executablePath = config.Steam.ExecutablePath
	} else {
		executablePath = filepath.Join(ServerInstallPath, config.Steam.ExecutablePath)
	}

	as.cmd = buildCommand(as.ctx, executablePath)
	as.cmd.Dir = ServerInstallPath

	if as.out == nil {
		as.out = newLogBuffer(MaxLogSizeBytes)
	}

	as.cmd.Stdout = as.out
	as.cmd.Stderr = as.out

	err = as.cmd.Start()

	if err != nil {
		as.cmd = nil
		return err
	}

	for _, command := range config.Server.RunOnStart {
		parts := strings.Split(command, " ")

		var cmd *exec.Cmd

		if len(parts) > 1 {
			cmd = buildCommand(as.ctx, parts[0], parts[1:]...)
		} else {
			cmd = buildCommand(as.ctx, parts[0])
		}

		cmd.Stdout = pluginsOutput
		cmd.Stderr = pluginsOutput
		cmd.Dir = wd

		err := cmd.Start()

		if err != nil {
			logrus.Errorf("Could not run extra command: %s, err: %s", command, err)
			continue
		}

		as.extraProcesses = append(as.extraProcesses, cmd)
	}

	go func() {
		_ = as.cmd.Wait()
		as.stopChildProcesses()
	}()

	return nil
}

func (as *AssettoServerProcess) stopChildProcesses() {
	for _, command := range as.extraProcesses {
		err := kill(command.Process)

		if err != nil {
			logrus.Errorf("Can't kill process: %d, err: %s", command.Process.Pid, err)
			continue
		}

		_ = command.Process.Release()
	}

	as.extraProcesses = make([]*exec.Cmd, 0)
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

func (as *AssettoServerProcess) EventType() ServerEventType {
	if championshipManager.activeChampionship != nil {
		return EventTypeChampionship
	} else {
		return EventTypeRace
	}
}

// Stop the assetto server.
func (as *AssettoServerProcess) Stop() error {
	if !as.IsRunning() {
		return nil
	}

	as.mutex.Lock()
	defer as.mutex.Unlock()

	err := kill(as.cmd.Process)

	if err != nil && !strings.Contains(err.Error(), "process already finished") {
		logrus.WithError(err).Errorf("Stopping server reported an error (continuing anyway...)")
	}

	as.stopChildProcesses()

	for {
		proc, err := ps.FindProcess(as.cmd.Process.Pid)

		if err != nil {
			logrus.WithError(err).Warnf("Could not find process: %d", as.cmd.Process.Pid)
		}

		if proc == nil {
			break
		}

		logrus.Debugf("Waiting for Assetto Corsa Server process to finish...")
		time.Sleep(time.Millisecond * 500)
	}

	as.cmd = nil
	as.doneCh <- struct{}{}

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

func newLogBuffer(maxSize int) *logBuffer {
	return &logBuffer{
		size: maxSize,
		buf:  new(bytes.Buffer),
	}
}

type logBuffer struct {
	buf *bytes.Buffer

	size int
}

func (lb *logBuffer) Write(p []byte) (n int, err error) {
	b := lb.buf.Bytes()

	if len(b) > lb.size {
		lb.buf = bytes.NewBuffer(b[len(b)-lb.size:])
	}

	return lb.buf.Write(p)
}

func (lb *logBuffer) String() string {
	return lb.buf.String()
}
