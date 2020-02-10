package main

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	servermanager "github.com/JustaPenguin/assetto-server-manager"
	"github.com/JustaPenguin/assetto-server-manager/cmd/server-manager/static"
	"github.com/JustaPenguin/assetto-server-manager/cmd/server-manager/views"
	"github.com/JustaPenguin/assetto-server-manager/internal/changelog"
	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/lorenzosaino/go-sysctl"
	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
	lua "github.com/yuin/gopher-lua"
)

var defaultAddress = "0.0.0.0:8772"

const (
	udpRealtimePosRefreshIntervalMin = 100
)

func init() {
	runtime.LockOSThread()
	servermanager.InitLogging()
}

func main() {
	config, err := servermanager.ReadConfig("config.yml")

	if err != nil {
		ServeHTTPWithError(defaultAddress, "Read configuration file (config.yml)", err)
		return
	}

	if config.Monitoring.Enabled {
		servermanager.InitMonitoring()
	}

	store, err := config.Store.BuildStore()

	if err != nil {
		ServeHTTPWithError(config.HTTP.Hostname, "Open server manager storage (bolt or json)", err)
		return
	}

	changes, err := changelog.LoadChangelog()

	if err != nil {
		ServeHTTPWithError(config.HTTP.Hostname, "Load changelog (internal error)", err)
		return
	}

	servermanager.Changelog = changes

	var templateLoader servermanager.TemplateLoader
	var filesystem http.FileSystem

	if os.Getenv("FILESYSTEM_HTML") == "true" {
		templateLoader = servermanager.NewFilesystemTemplateLoader("views")
		filesystem = http.Dir("static")
	} else {
		templateLoader = &views.TemplateLoader{}
		filesystem = static.FS(false)
	}

	resolver, err := servermanager.NewResolver(templateLoader, os.Getenv("FILESYSTEM_HTML") == "true", store)

	if err != nil {
		ServeHTTPWithError(config.HTTP.Hostname, "Initialise resolver (internal error)", err)
		return
	}
	servermanager.SetAssettoInstallPath(config.Steam.InstallPath)

	err = servermanager.InstallAssettoCorsaServer(config.Steam.Username, config.Steam.Password, config.Steam.ForceUpdate)

	if err != nil {
		ServeHTTPWithError(defaultAddress, "Install assetto corsa server with steamcmd. Likely you do not have steamcmd installed correctly.", err)
		return
	}

	if config.LiveMap.IsEnabled() {
		if config.LiveMap.IntervalMs < udpRealtimePosRefreshIntervalMin {
			udp.RealtimePosIntervalMs = udpRealtimePosRefreshIntervalMin
		} else {
			udp.RealtimePosIntervalMs = config.LiveMap.IntervalMs
		}

		if runtime.GOOS == "linux" {
			// check known kernel net memory restrictions. if they're lower than the recommended
			// values, then print out explaining how to increase them
			memValues := []string{"net.core.rmem_max", "net.core.rmem_default", "net.core.wmem_max", "net.core.wmem_default"}

			for _, val := range memValues {
				checkMemValue(val)
			}
		}
	}

	if config.Lua.Enabled && servermanager.Premium() {
		luaPath := os.Getenv("LUA_PATH")

		newPath, err := filepath.Abs("./plugins/?.lua")

		if err != nil {
			logrus.WithError(err).Error("Couldn't get absolute path for /plugins folder")
		} else {
			if luaPath != "" {
				luaPath = luaPath + ";" + newPath
			} else {
				luaPath = newPath
			}

			err = os.Setenv("LUA_PATH", luaPath)

			if err != nil {
				logrus.WithError(err).Error("Couldn't automatically set Lua path, lua will not run! Try setting the environment variable LUA_PATH manually.")
			}
		}

		servermanager.Lua = lua.NewState()
		defer servermanager.Lua.Close()

		servermanager.InitLua(resolver.ResolveRaceControl())
	}

	err = servermanager.InitWithResolver(resolver)

	if err != nil {
		ServeHTTPWithError(config.HTTP.Hostname, "Initialise server manager (internal error)", err)
		return
	}

	listener, err := net.Listen("tcp", config.HTTP.Hostname)

	if err != nil {
		ServeHTTPWithError(defaultAddress, "Listen on hostname "+config.HTTP.Hostname+". Likely the port has already been taken by another application", err)
		return
	}

	logrus.Infof("starting assetto server manager on: %s", config.HTTP.Hostname)

	if !config.Server.DisableWindowsBrowserOpen && runtime.GOOS == "windows" {
		_ = browser.OpenURL("http://" + strings.Replace(config.HTTP.Hostname, "0.0.0.0", "127.0.0.1", 1))
	}

	router := resolver.ResolveRouter(filesystem)

	if err := http.Serve(listener, router); err != nil {
		logrus.Fatal(err)
	}
}

const udpBufferRecommendedSize = uint64(2e6) // 2MB

func checkMemValue(key string) {
	val, err := sysctlAsUint64(key)

	if err != nil {
		logrus.WithError(err).Errorf("Could not check sysctl val: %s", key)
		return
	}

	if val < udpBufferRecommendedSize {
		d := color.New(color.FgRed)
		red := d.PrintfFunc()
		redln := d.PrintlnFunc()

		redln()
		redln("-------------------------------------------------------------------")
		redln("                          W A R N I N G")
		redln("-------------------------------------------------------------------")

		red("System %s value is too small! UDP messages are \n", key)
		redln("more likely to be lost and the stability of various Server Manager")
		redln("systems will be greatly affected.")
		redln()

		red("Your current value is %s. We recommend a value of %s for a \n", humanize.Bytes(val), humanize.Bytes(udpBufferRecommendedSize))
		redln("more consistent operation.")
		redln()

		red("You can do this with the command:\n\t sysctl -w %s=%d\n", key, udpBufferRecommendedSize)
		redln()

		redln("More information can be found on sysctl variables here:\n\t https://www.cyberciti.biz/faq/howto-set-sysctl-variables/")
	}
}

func sysctlAsUint64(val string) (uint64, error) {
	val, err := sysctl.Get(val)

	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(val, 10, 0)
}
