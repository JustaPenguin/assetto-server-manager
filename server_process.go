package servermanager

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"
	"github.com/mitchellh/go-ps"

	"github.com/sirupsen/logrus"
)

const MaxLogSizeBytes = 1e6

type ServerProcess interface {
	Logs() string
	Start(cfg ServerConfig, entryList EntryList, forwardingAddress string, forwardListenPort int, event RaceEvent) error
	Stop() error
	Restart() error
	IsRunning() bool
	Event() RaceEvent
	UDPCallback(message udp.Message)
	SendUDPMessage(message udp.Message) error
	GetServerConfig() ServerConfig

	Done() <-chan struct{}
}

var ErrServerAlreadyRunning = errors.New("servermanager: assetto corsa server is already running")

// AssettoServerProcess manages the assetto corsa server process.
type AssettoServerProcess struct {
	contentManagerWrapper *ContentManagerWrapper

	cmd *exec.Cmd

	out   *logBuffer
	mutex sync.Mutex

	ctx context.Context
	cfn context.CancelFunc

	doneCh chan struct{}
	store  Store

	extraProcesses []*exec.Cmd

	serverConfig      ServerConfig
	entryList         EntryList
	forwardingAddress string
	forwardListenPort int
	udpServerConn     *udp.AssettoServerUDP
	udpStatusMutex    sync.Mutex
	callbackFunc      udp.CallbackFunc
	event             RaceEvent
}

func NewAssettoServerProcess(callbackFunc udp.CallbackFunc, store Store, contentManagerWrapper *ContentManagerWrapper) *AssettoServerProcess {
	ctx, cfn := context.WithCancel(context.Background())

	return &AssettoServerProcess{
		ctx:                   ctx,
		cfn:                   cfn,
		doneCh:                make(chan struct{}),
		callbackFunc:          callbackFunc,
		contentManagerWrapper: contentManagerWrapper,
		store:                 store,
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
func (as *AssettoServerProcess) Start(cfg ServerConfig, entryList EntryList, forwardingAddress string, forwardListenPort int, event RaceEvent) error {
	if as.IsRunning() {
		return ErrServerAlreadyRunning
	}

	as.mutex.Lock()
	defer as.mutex.Unlock()

	logrus.Debugf("Starting assetto server process")

	as.serverConfig = cfg
	as.entryList = entryList
	as.forwardingAddress = forwardingAddress
	as.forwardListenPort = forwardListenPort
	as.event = event

	if err := as.startUDPListener(); err != nil {
		return err
	}

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

	if cfg.GlobalServerConfig.EnableContentManagerWrapper == 1 && cfg.GlobalServerConfig.ContentManagerWrapperPort > 0 {
		go func() {
			err := as.contentManagerWrapper.Start(as, cfg.GlobalServerConfig.ContentManagerWrapperPort, cfg, entryList, event)

			if err != nil {
				logrus.WithError(err).Error("Could not start Content Manager wrapper server")
			}
		}()
	}

	if strackerOptions, err := as.store.LoadStrackerOptions(); err == nil && strackerOptions.EnableStracker && IsStrackerInstalled() {
		if as.forwardListenPort >= 0 && as.forwardingAddress != "" || strings.Contains(as.forwardingAddress, ":") {
			strackerOptions.ACPlugin.SendPort = as.forwardListenPort
			strackerOptions.ACPlugin.ReceivePort = formValueAsInt(strings.Split(as.forwardingAddress, ":")[1])

			err = strackerOptions.Write()

			if err != nil {
				return err
			}

			err = as.startPlugin(wd, &CommandPlugin{
				Executable: StrackerExecutablePath(),
				Arguments: []string{
					"--stracker_ini",
					filepath.Join(StrackerFolderPath(), strackerConfigIniFilename),
				},
			})

			if err != nil {
				return err
			}

			logrus.Infof("Started stracker. Listening for ptracker connections on port %d", strackerOptions.InstanceConfiguration.ListeningPort)
		} else {
			logrus.WithError(ErrStrackerConfigurationRequiresUDPPluginConfiguration).Error("Please check your server configuration")
		}
	}

	for _, plugin := range config.Server.Plugins {
		err = as.startPlugin(wd, plugin)

		if err != nil {
			logrus.WithError(err).Errorf("Could not run extra command: %s", plugin.String())
		}
	}

	if len(config.Server.RunOnStart) > 0 {
		logrus.Warnf("Use of run_on_start in config.yml is deprecated. Please use 'plugins' instead")
	}

	for _, command := range config.Server.RunOnStart {
		err = as.startChildProcess(wd, command)

		if err != nil {
			logrus.WithError(err).Errorf("Could not run extra command: %s", command)
		}
	}

	go func() {
		_ = as.cmd.Wait()

		loopNum := 0

		for {
			if loopNum > 50 {
				break
			}

			proc, err := ps.FindProcess(as.cmd.Process.Pid)

			if err != nil {
				logrus.WithError(err).Warnf("Could not find process: %d", as.cmd.Process.Pid)
			}

			if proc == nil {
				break
			}

			logrus.Debugf("Waiting for Assetto Corsa Server process to finish...")
			time.Sleep(time.Millisecond * 500)
			loopNum++
		}

		logrus.Infof("Detected server shutdown. Closing child processes")

		as.stopChildProcesses()
		as.closeUDPConnection()

		as.cmd = nil
		go func() {
			as.doneCh <- struct{}{}
		}()
	}()

	return nil
}

func (as *AssettoServerProcess) startPlugin(wd string, plugin *CommandPlugin) error {
	commandFullPath, err := filepath.Abs(plugin.Executable)

	if err != nil {
		as.cmd = nil
		return err
	}

	cmd := buildCommand(as.ctx, commandFullPath, plugin.Arguments...)

	pluginDir, err := filepath.Abs(filepath.Dir(commandFullPath))

	if err != nil {
		logrus.WithError(err).Warnf("Could not determine plugin directory. Setting working dir to: %s", wd)
		pluginDir = wd
	}

	cmd.Stdout = pluginsOutput
	cmd.Stderr = pluginsOutput

	cmd.Dir = pluginDir

	err = cmd.Start()

	if err != nil {
		return err
	}

	as.extraProcesses = append(as.extraProcesses, cmd)

	return nil
}

// Deprecated: use startPlugin instead
func (as *AssettoServerProcess) startChildProcess(wd string, command string) error {
	// BUG(cj): splitting commands on spaces breaks child processes that have a space in their path name
	parts := strings.Split(command, " ")

	if len(parts) == 0 {
		return nil
	}

	commandFullPath, err := filepath.Abs(parts[0])

	if err != nil {
		as.cmd = nil
		return err
	}

	var cmd *exec.Cmd

	if len(parts) > 1 {
		cmd = buildCommand(as.ctx, commandFullPath, parts[1:]...)
	} else {
		cmd = buildCommand(as.ctx, commandFullPath)
	}

	pluginDir, err := filepath.Abs(filepath.Dir(commandFullPath))

	if err != nil {
		logrus.WithError(err).Warnf("Could not determine plugin directory. Setting working dir to: %s", wd)
		pluginDir = wd
	}

	cmd.Stdout = pluginsOutput
	cmd.Stderr = pluginsOutput

	cmd.Dir = pluginDir

	err = cmd.Start()

	if err != nil {
		return err
	}

	as.extraProcesses = append(as.extraProcesses, cmd)

	return nil
}

func (as *AssettoServerProcess) closeUDPConnection() {
	as.udpStatusMutex.Lock()
	defer as.udpStatusMutex.Unlock()

	if as.udpServerConn == nil {
		return
	}

	logrus.Debugf("Closing UDP connection")

	err := as.udpServerConn.Close()

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't close UDP connection")
	}

	as.udpServerConn = nil
}

func (as *AssettoServerProcess) startUDPListener() error {
	as.udpStatusMutex.Lock()
	defer as.udpStatusMutex.Unlock()

	if as.udpServerConn != nil {
		return nil
	}

	var err error

	host, portStr, err := net.SplitHostPort(as.serverConfig.GlobalServerConfig.FreeUDPPluginAddress)

	if err != nil {
		return err
	}

	port, err := strconv.ParseInt(portStr, 10, 0)

	if err != nil {
		return err
	}

	as.udpServerConn, err = udp.NewServerClient(host, int(port), as.serverConfig.GlobalServerConfig.FreeUDPPluginLocalPort, true, as.forwardingAddress, as.forwardListenPort, as.UDPCallback)

	if err != nil {
		return err
	}

	return nil
}

func (as *AssettoServerProcess) UDPCallback(message udp.Message) {
	panicCapture(func() {
		as.callbackFunc(message)
	})
}

var ErrNoOpenUDPConnection = errors.New("servermanager: no open UDP connection found")

func (as *AssettoServerProcess) SendUDPMessage(message udp.Message) error {
	if as.udpServerConn == nil {
		return ErrNoOpenUDPConnection
	}

	return as.udpServerConn.SendMessage(message)
}

func (as *AssettoServerProcess) stopChildProcesses() {
	if as.serverConfig.GlobalServerConfig.EnableContentManagerWrapper == 1 && as.serverConfig.GlobalServerConfig.ContentManagerWrapperPort > 0 {
		as.contentManagerWrapper.Stop()
	}

	for _, command := range as.extraProcesses {
		err := kill(command.Process)

		if err != nil {
			logrus.WithError(err).Errorf("Can't kill process: %d", command.Process.Pid)
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

	return as.Start(as.serverConfig, as.entryList, as.forwardingAddress, as.forwardListenPort, as.event)
}

// IsRunning of the server. returns true if running
func (as *AssettoServerProcess) IsRunning() bool {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	return as.cmd != nil && as.cmd.Process != nil
}

func (as *AssettoServerProcess) Event() RaceEvent {
	if as.event == nil {
		return QuickRace{}
	}

	return as.event
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

	return nil
}

func (as *AssettoServerProcess) GetServerConfig() ServerConfig {
	return as.serverConfig
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
