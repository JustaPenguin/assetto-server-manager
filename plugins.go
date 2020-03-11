package servermanager

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

type PluginManager struct {
	store Store

	running []*exec.Cmd
}

func NewPluginManager(store Store) *PluginManager {
	return &PluginManager{
		store: store,
	}
}

func (pm *PluginManager) StartPlugins() error {
	serverOptions, err := pm.store.LoadServerOptions()

	if err != nil {
		return err
	}

	strackerOptions, err := pm.store.LoadStrackerOptions()
	strackerEnabled := err == nil && strackerOptions.EnableStracker && IsStrackerInstalled()

	kissMyRankOptions, err := pm.store.LoadKissMyRankOptions()
	kissMyRankEnabled := err == nil && kissMyRankOptions.EnableKissMyRank && IsKissMyRankInstalled()

	udpPluginPortsSetup := serverOptions.UDPPluginLocalPort >= 0 && serverOptions.UDPPluginAddress != "" || strings.Contains(serverOptions.UDPPluginAddress, ":")

	if (strackerEnabled || kissMyRankEnabled) && !udpPluginPortsSetup {
		logrus.WithError(ErrPluginConfigurationRequiresUDPPortSetup).Error("Please check your server configuration")
	}

	wd, err := os.Getwd()

	if err != nil {
		return err
	}

	if strackerEnabled && strackerOptions != nil && udpPluginPortsSetup {
		strackerOptions.InstanceConfiguration.ACServerConfigIni = filepath.Join(ServerInstallPath, "cfg", serverConfigIniPath)
		strackerOptions.InstanceConfiguration.ACServerWorkingDir = ServerInstallPath
		strackerOptions.ACPlugin.SendPort = serverOptions.UDPPluginLocalPort
		strackerOptions.ACPlugin.ReceivePort = formValueAsInt(strings.Split(serverOptions.UDPPluginAddress, ":")[1])

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

		err = pm.startPlugin(wd, &CommandPlugin{
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
		kissMyRankOptions.ACServerResultsBasePath = ServerInstallPath

		if strackerEnabled {
			// stracker is enabled, use its forwarding port
			logrus.Infof("sTracker and KissMyRank both enabled. Using plugin forwarding method: [Server Manager] <-> [sTracker] <-> [KissMyRank]")
			kissMyRankOptions.ACServerPluginLocalPort = strackerOptions.ACPlugin.ProxyPluginLocalPort
			kissMyRankOptions.ACServerPluginAddressPort = strackerOptions.ACPlugin.ProxyPluginPort
		} else {
			// stracker is disabled, use our forwarding port
			kissMyRankOptions.ACServerPluginLocalPort = serverOptions.UDPPluginLocalPort
			kissMyRankOptions.ACServerPluginAddressPort = formValueAsInt(strings.Split(serverOptions.UDPPluginAddress, ":")[1])
		}

		if err := kissMyRankOptions.Write(); err != nil {
			return err
		}

		err = pm.startPlugin(wd, &CommandPlugin{
			Executable: KissMyRankExecutablePath(),
		})

		if err != nil {
			return err
		}

		logrus.Infof("Started KissMyRank")
	}

	return nil
}

func (pm *PluginManager) startPlugin(wd string, plugin *CommandPlugin) error {
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

	if err := cmd.Start(); err != nil {
		return err
	}

	pm.running = append(pm.running, cmd)

	return nil
}

func (pm *PluginManager) StopPlugins() {
	for _, plugin := range pm.running {
		plugin := plugin

		if err := plugin.Process.Signal(os.Interrupt); err != nil {
			logrus.WithError(err).Debug("Could not stop plugin")
		}
	}

	pm.running = []*exec.Cmd{}
}
