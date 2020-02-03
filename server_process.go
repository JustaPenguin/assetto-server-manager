package servermanager

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"

	"github.com/sirupsen/logrus"
)

const MaxLogSizeBytes = 1e6

type ServerProcess interface {
	Start(event RaceEvent, udpPluginAddress string, udpPluginLocalPort int, forwardingAddress string, forwardListenPort int) error
	Stop() error
	Restart() error
	IsRunning() bool
	Event() RaceEvent
	UDPCallback(message udp.Message)
	SendUDPMessage(message udp.Message) error
	NotifyDone(chan struct{})
	Logs() string
}

// AssettoServerProcess manages the Assetto Corsa Server process.
type AssettoServerProcess struct {
	store                 Store
	contentManagerWrapper *ContentManagerWrapper

	start                 chan RaceEvent
	started, stopped, run chan error
	notifyDoneChs         []chan struct{}

	ctx context.Context
	cfn context.CancelFunc

	logBuffer *logBuffer

	raceEvent      RaceEvent
	cmd            *exec.Cmd
	mutex          sync.Mutex
	extraProcesses []*exec.Cmd

	logFile, errorLogFile io.WriteCloser

	// udp
	callbackFunc       udp.CallbackFunc
	udpServerConn      *udp.AssettoServerUDP
	udpPluginAddress   string
	udpPluginLocalPort int
	forwardingAddress  string
	forwardListenPort  int
}

func NewAssettoServerProcess(callbackFunc udp.CallbackFunc, store Store, contentManagerWrapper *ContentManagerWrapper) *AssettoServerProcess {
	sp := &AssettoServerProcess{
		start:                 make(chan RaceEvent),
		started:               make(chan error),
		stopped:               make(chan error),
		run:                   make(chan error),
		logBuffer:             newLogBuffer(MaxLogSizeBytes),
		callbackFunc:          callbackFunc,
		store:                 store,
		contentManagerWrapper: contentManagerWrapper,
	}

	go sp.loop()

	return sp
}

func (sp *AssettoServerProcess) UDPCallback(message udp.Message) {
	panicCapture(func() {
		sp.callbackFunc(message)
	})
}

func (sp *AssettoServerProcess) Start(event RaceEvent, udpPluginAddress string, udpPluginLocalPort int, forwardingAddress string, forwardListenPort int) error {
	sp.mutex.Lock()
	sp.udpPluginAddress = udpPluginAddress
	sp.udpPluginLocalPort = udpPluginLocalPort
	sp.forwardingAddress = forwardingAddress
	sp.forwardListenPort = forwardListenPort
	sp.mutex.Unlock()

	sp.start <- event

	return <-sp.started
}

var ErrPluginConfigurationRequiresUDPPortSetup = errors.New("servermanager: kissmyrank and stracker configuration requires UDP plugin configuration in Server Options")

func (sp *AssettoServerProcess) IsRunning() bool {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	return sp.raceEvent != nil
}

var ErrServerProcessTimeout = errors.New("servermanager: server process did not stop even after manual kill. please check your server configuration")

func (sp *AssettoServerProcess) Stop() error {
	if !sp.IsRunning() {
		return nil
	}

	timeout := time.After(time.Second * 10)
	errCh := make(chan error)

	go func() {
		select {
		case err := <-sp.stopped:
			errCh <- err
			return
		case <-timeout:
			errCh <- ErrServerProcessTimeout
			return
		}
	}()

	if err := kill(sp.cmd.Process); err != nil {
		logrus.WithError(err).Error("Could not forcibly kill command")
	}

	sp.cfn()

	return <-errCh
}

func (sp *AssettoServerProcess) Restart() error {
	running := sp.IsRunning()

	sp.mutex.Lock()
	raceEvent := sp.raceEvent
	udpPluginAddress := sp.udpPluginAddress
	udpLocalPluginPort := sp.udpPluginLocalPort
	forwardingAddress := sp.forwardingAddress
	forwardListenPort := sp.forwardListenPort
	sp.mutex.Unlock()

	if running {
		if err := sp.Stop(); err != nil {
			return err
		}
	}

	return sp.Start(raceEvent, udpPluginAddress, udpLocalPluginPort, forwardingAddress, forwardListenPort)
}

func (sp *AssettoServerProcess) loop() {
	for {
		select {
		case err := <-sp.run:
			if err != nil {
				logrus.WithError(err).Warn("acServer process ended with error. If everything seems fine, you can safely ignore this error.")
			}

			select {
			case sp.stopped <- sp.onStop():
			default:
			}
		case raceEvent := <-sp.start:
			if sp.IsRunning() {
				if err := sp.Stop(); err != nil {
					sp.started <- err
					break
				}
			}

			sp.started <- sp.startRaceEvent(raceEvent)
		}
	}
}

func (sp *AssettoServerProcess) startRaceEvent(raceEvent RaceEvent) error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	logrus.Infof("Starting Server Process with event: %s", describeRaceEvent(raceEvent))
	var executablePath string

	if filepath.IsAbs(config.Steam.ExecutablePath) {
		executablePath = config.Steam.ExecutablePath
	} else {
		executablePath = filepath.Join(ServerInstallPath, config.Steam.ExecutablePath)
	}

	serverOptions, err := sp.store.LoadServerOptions()

	if err != nil {
		return err
	}

	sp.ctx, sp.cfn = context.WithCancel(context.Background())
	sp.cmd = buildCommand(sp.ctx, executablePath)
	sp.cmd.Dir = ServerInstallPath

	var logOutput io.Writer
	var errorOutput io.Writer

	if serverOptions.LogACServerOutputToFile {
		logDirectory := filepath.Join(ServerInstallPath, "logs", "session")
		errorDirectory := filepath.Join(ServerInstallPath, "logs", "error")

		if err := os.MkdirAll(logDirectory, 0755); err != nil {
			return err
		}

		if err := os.MkdirAll(errorDirectory, 0755); err != nil {
			return err
		}

		if err := sp.deleteOldLogFiles(serverOptions.NumberOfACServerLogsToKeep); err != nil {
			return err
		}

		timestamp := time.Now().Format("2006-02-01_15-04-05")

		sp.logFile, err = os.Create(filepath.Join(logDirectory, "output_"+timestamp+".log"))

		if err != nil {
			return err
		}

		sp.errorLogFile, err = os.Create(filepath.Join(errorDirectory, "error_"+timestamp+".log"))

		if err != nil {
			return err
		}

		logOutput = io.MultiWriter(sp.logBuffer, sp.logFile)
		errorOutput = io.MultiWriter(sp.logBuffer, sp.errorLogFile)
	} else {
		logOutput = sp.logBuffer
		errorOutput = sp.logBuffer
	}

	sp.cmd.Stdout = logOutput
	sp.cmd.Stderr = errorOutput

	if err := sp.startUDPListener(); err != nil {
		return err
	}

	wd, err := os.Getwd()

	if err != nil {
		return err
	}

	sp.raceEvent = raceEvent

	go func() {
		sp.run <- sp.cmd.Run()
	}()

	if serverOptions.EnableContentManagerWrapper == 1 && serverOptions.ContentManagerWrapperPort > 0 {
		go panicCapture(func() {
			err := sp.contentManagerWrapper.Start(serverOptions.ContentManagerWrapperPort, sp.raceEvent)

			if err != nil {
				logrus.WithError(err).Error("Could not start Content Manager wrapper server")
			}
		})
	}

	strackerOptions, err := sp.store.LoadStrackerOptions()
	strackerEnabled := err == nil && strackerOptions.EnableStracker && IsStrackerInstalled()

	kissMyRankOptions, err := sp.store.LoadKissMyRankOptions()
	kissMyRankEnabled := err == nil && kissMyRankOptions.EnableKissMyRank && IsKissMyRankInstalled()

	udpPluginPortsSetup := sp.forwardListenPort >= 0 && sp.forwardingAddress != "" || strings.Contains(sp.forwardingAddress, ":")

	if (strackerEnabled || kissMyRankEnabled) && !udpPluginPortsSetup {
		logrus.WithError(ErrPluginConfigurationRequiresUDPPortSetup).Error("Please check your server configuration")
	}

	if strackerEnabled && strackerOptions != nil && udpPluginPortsSetup {
		strackerOptions.InstanceConfiguration.ACServerConfigIni = filepath.Join(ServerInstallPath, "cfg", serverConfigIniPath)
		strackerOptions.InstanceConfiguration.ACServerWorkingDir = ServerInstallPath
		strackerOptions.ACPlugin.SendPort = sp.forwardListenPort
		strackerOptions.ACPlugin.ReceivePort = formValueAsInt(strings.Split(sp.forwardingAddress, ":")[1])

		if kissMyRankEnabled {
			// kissmyrank uses stracker's forwarding to chain the plugins. make sure that it is set up.
			if strackerOptions.ACPlugin.ProxyPluginLocalPort <= 0 {
				strackerOptions.ACPlugin.ProxyPluginLocalPort, err = FreeUDPPort()

				if err != nil {
					return err
				}
			}

			for strackerOptions.ACPlugin.ProxyPluginPort <= 0 || strackerOptions.ACPlugin.ProxyPluginPort == strackerOptions.ACPlugin.ProxyPluginLocalPort {
				strackerOptions.ACPlugin.ProxyPluginPort, err = FreeUDPPort()

				if err != nil {
					return err
				}
			}
		}

		if err := strackerOptions.Write(); err != nil {
			return err
		}

		err = sp.startPlugin(wd, &CommandPlugin{
			Executable: StrackerExecutablePath(),
			Arguments: []string{
				"--stracker_ini",
				filepath.Join(StrackerFolderPath(), strackerConfigIniFilename),
			},
		})

		if err != nil {
			return err
		}

		logrus.Infof("Started sTracker. Listening for pTracker connections on port %d", strackerOptions.InstanceConfiguration.ListeningPort)
	}

	if kissMyRankEnabled && kissMyRankOptions != nil && udpPluginPortsSetup {
		if err := fixKissMyRankExecutablePermissions(); err != nil {
			return err
		}

		kissMyRankOptions.ACServerIP = "127.0.0.1"
		kissMyRankOptions.ACServerHTTPPort = serverOptions.HTTPPort
		kissMyRankOptions.UpdateInterval = config.LiveMap.IntervalMs
		kissMyRankOptions.ACServerResultsBasePath = filepath.Join(ServerInstallPath, "results")

		raceConfig := sp.raceEvent.GetRaceConfig()
		entryList := sp.raceEvent.GetEntryList()

		kissMyRankOptions.MaxPlayers = raceConfig.MaxClients

		if len(entryList) > kissMyRankOptions.MaxPlayers {
			kissMyRankOptions.MaxPlayers = len(entryList)
		}

		if strackerEnabled {
			// stracker is enabled, use its forwarding port
			logrus.Infof("sTracker and KissMyRank both enabled. Using plugin forwarding method: [Server Manager] <-> [sTracker] <-> [KissMyRank]")
			kissMyRankOptions.ACServerPluginLocalPort = strackerOptions.ACPlugin.ProxyPluginLocalPort
			kissMyRankOptions.ACServerPluginAddressPort = strackerOptions.ACPlugin.ProxyPluginPort
		} else {
			// stracker is disabled, use our forwarding port
			kissMyRankOptions.ACServerPluginLocalPort = sp.forwardListenPort
			kissMyRankOptions.ACServerPluginAddressPort = formValueAsInt(strings.Split(sp.forwardingAddress, ":")[1])
		}

		if err := kissMyRankOptions.Write(); err != nil {
			return err
		}

		err = sp.startPlugin(wd, &CommandPlugin{
			Executable: KissMyRankExecutablePath(),
		})

		if err != nil {
			return err
		}

		logrus.Infof("Started KissMyRank")
	}

	for _, plugin := range config.Server.Plugins {
		err = sp.startPlugin(wd, plugin)

		if err != nil {
			logrus.WithError(err).Errorf("Could not run extra command: %s", plugin.String())
		}
	}

	if len(config.Server.RunOnStart) > 0 {
		logrus.Warnf("Use of run_on_start in config.yml is deprecated. Please use 'plugins' instead")

		for _, command := range config.Server.RunOnStart {
			err = sp.startChildProcess(wd, command)

			if err != nil {
				logrus.WithError(err).Errorf("Could not run extra command: %s", command)
			}
		}
	}

	return nil
}

func (sp *AssettoServerProcess) deleteOldLogFiles(numFilesToKeep int) error {
	if numFilesToKeep <= 0 {
		return nil
	}

	tidyFunc := func(directory string) error {
		logFiles, err := ioutil.ReadDir(directory)

		if err != nil {
			return err
		}

		if len(logFiles) >= numFilesToKeep {
			sort.Slice(logFiles, func(i, j int) bool {
				return logFiles[i].ModTime().After(logFiles[j].ModTime())
			})

			for _, f := range logFiles[numFilesToKeep-1:] {
				if err := os.Remove(filepath.Join(directory, f.Name())); err != nil {
					return err
				}
			}

			logrus.Debugf("Successfully cleared %d log files from %s", len(logFiles[numFilesToKeep-1:]), directory)
		}

		return nil
	}

	logDirectory := filepath.Join(ServerInstallPath, "logs", "session")
	errorDirectory := filepath.Join(ServerInstallPath, "logs", "error")

	if err := tidyFunc(logDirectory); err != nil {
		return err
	}

	if err := tidyFunc(errorDirectory); err != nil {
		return err
	}

	return nil
}

func (sp *AssettoServerProcess) onStop() error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	logrus.Debugf("Server stopped. Stopping UDP listener and child processes.")

	sp.raceEvent = nil

	if err := sp.stopUDPListener(); err != nil {
		return err
	}

	sp.stopChildProcesses()

	for _, doneCh := range sp.notifyDoneChs {
		select {
		case doneCh <- struct{}{}:
		default:
		}
	}

	if sp.logFile != nil {
		if err := sp.logFile.Close(); err != nil {
			return err
		}

		sp.logFile = nil
	}

	if sp.errorLogFile != nil {
		if err := sp.errorLogFile.Close(); err != nil {
			return err
		}

		sp.errorLogFile = nil
	}

	return nil
}

func (sp *AssettoServerProcess) Logs() string {
	return sp.logBuffer.String()
}

func (sp *AssettoServerProcess) Event() RaceEvent {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	if sp.raceEvent == nil {
		return QuickRace{}
	}

	return sp.raceEvent
}

var ErrNoOpenUDPConnection = errors.New("servermanager: no open UDP connection found")

func (sp *AssettoServerProcess) SendUDPMessage(message udp.Message) error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	if sp.udpServerConn == nil {
		return ErrNoOpenUDPConnection
	}

	return sp.udpServerConn.SendMessage(message)
}

func (sp *AssettoServerProcess) NotifyDone(ch chan struct{}) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	sp.notifyDoneChs = append(sp.notifyDoneChs, ch)
}

func (sp *AssettoServerProcess) startPlugin(wd string, plugin *CommandPlugin) error {
	commandFullPath, err := filepath.Abs(plugin.Executable)

	if err != nil {
		return err
	}

	ctx := context.Background()

	cmd := buildCommand(ctx, commandFullPath, plugin.Arguments...)

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

	sp.extraProcesses = append(sp.extraProcesses, cmd)

	return nil
}

// Deprecated: use startPlugin instead
func (sp *AssettoServerProcess) startChildProcess(wd string, command string) error {
	// BUG(cj): splitting commands on spaces breaks child processes that have a space in their path name
	parts := strings.Split(command, " ")

	if len(parts) == 0 {
		return nil
	}

	commandFullPath, err := filepath.Abs(parts[0])

	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	ctx := context.Background()

	if len(parts) > 1 {
		cmd = buildCommand(ctx, commandFullPath, parts[1:]...)
	} else {
		cmd = buildCommand(ctx, commandFullPath)
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

	sp.extraProcesses = append(sp.extraProcesses, cmd)

	return nil
}

func (sp *AssettoServerProcess) stopChildProcesses() {
	sp.contentManagerWrapper.Stop()

	for _, command := range sp.extraProcesses {
		err := kill(command.Process)

		if err != nil {
			logrus.WithError(err).Errorf("Can't kill process: %d", command.Process.Pid)
			continue
		}

		_ = command.Process.Release()
	}

	sp.extraProcesses = make([]*exec.Cmd, 0)
}

func (sp *AssettoServerProcess) startUDPListener() error {
	var err error

	host, portStr, err := net.SplitHostPort(sp.udpPluginAddress)

	if err != nil {
		return err
	}

	port, err := strconv.ParseInt(portStr, 10, 0)

	if err != nil {
		return err
	}

	sp.udpServerConn, err = udp.NewServerClient(host, int(port), sp.udpPluginLocalPort, true, sp.forwardingAddress, sp.forwardListenPort, sp.UDPCallback)

	if err != nil {
		return err
	}

	return nil
}

func (sp *AssettoServerProcess) stopUDPListener() error {
	return sp.udpServerConn.Close()
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

	mutex sync.Mutex
}

func (lb *logBuffer) Write(p []byte) (n int, err error) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	b := lb.buf.Bytes()

	if len(b) > lb.size {
		lb.buf = bytes.NewBuffer(b[len(b)-lb.size:])
	}

	return lb.buf.Write(p)
}

func (lb *logBuffer) String() string {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	return lb.buf.String()
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
