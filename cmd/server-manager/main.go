package main

import (
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/cj123/assetto-server-manager"
	"github.com/cj123/assetto-server-manager/cmd/server-manager/static"
	"github.com/cj123/assetto-server-manager/cmd/server-manager/views"
	"github.com/cj123/assetto-server-manager/pkg/udp"
	"github.com/cj123/assetto-server-manager/pkg/udp/replay"

	"github.com/etcd-io/bbolt"
	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
)

var debug = os.Getenv("DEBUG") == "true"

var defaultAddress = "0.0.0.0:8772"

func init() {
	runtime.LockOSThread()
}

func main() {
	config, err := servermanager.ReadConfig("config.yml")

	if err != nil {
		ServeHTTPWithError(defaultAddress, "Read configuration file (config.yml)", err)
		return
	}

	store, err := config.Store.BuildStore()

	if err != nil {
		ServeHTTPWithError(config.HTTP.Hostname, "Open server manager storage (bolt or json)", err)
		return
	}

	servermanager.InitWithStore(store)
	servermanager.SetAssettoInstallPath(config.Steam.InstallPath)

	err = servermanager.InstallAssettoCorsaServer(config.Steam.Username, config.Steam.Password, config.Steam.ForceUpdate)

	if err != nil {
		ServeHTTPWithError(defaultAddress, "Install assetto corsa server with steamcmd. Likely you do not have steamcmd installed correctly.", err)
		return
	}

	var templateLoader servermanager.TemplateLoader
	var filesystem http.FileSystem

	if os.Getenv("FILESYSTEM_HTML") == "true" {
		templateLoader = servermanager.NewFilesystemTemplateLoader("views")
		filesystem = http.Dir("static")
	} else {
		templateLoader = &views.TemplateLoader{}
		filesystem = static.FS(false)
	}

	if config.LiveMap.IsEnabled() {
		udp.RealtimePosIntervalMs = config.LiveMap.IntervalMs
	}

	servermanager.ViewRenderer, err = servermanager.NewRenderer(templateLoader, os.Getenv("FILESYSTEM_HTML") == "true")

	if err != nil {
		ServeHTTPWithError(config.HTTP.Hostname, "Initialise view renderer (internal error)", err)
		return
	}

	go servermanager.LoopRaces()
	err = servermanager.InitialiseScheduledCustomRaces()

	if err != nil {
		logrus.Errorf("couldn't initialise scheduled races, err: %s", err)
	}

	err = servermanager.InitialiseScheduledChampionshipEvents()

	if err != nil {
		logrus.Errorf("couldn't initialise scheduled championship events, err: %s", err)
	}

	//go startUDPReplay("./assetto/session-logs/practiceloop_all.db")

	listener, err := net.Listen("tcp", config.HTTP.Hostname)

	if err != nil {
		ServeHTTPWithError(defaultAddress, "Listen on hostname "+config.HTTP.Hostname+". Likely the port has already been taken by another application", err)
		return
	}

	logrus.Infof("starting assetto server manager on: %s", config.HTTP.Hostname)

	if runtime.GOOS == "windows" {
		_ = browser.OpenURL("http://" + strings.Replace(config.HTTP.Hostname, "0.0.0.0", "127.0.0.1", 1))
	}

	if err := http.Serve(listener, servermanager.Router(filesystem)); err != nil {
		logrus.Fatal(err)
	}
}

func startUDPReplay(file string) {
	time.Sleep(time.Second * 20)

	db, err := bbolt.Open(file, 0644, nil)

	if err != nil {
		logrus.WithError(err).Error("Could not open bolt")
	}

	err = replay.ReplayUDPMessages(db, 10, func(response udp.Message) {
		servermanager.LiveTimingCallback(response)
		servermanager.LiveMapCallback(response)
		servermanager.LoopCallback(response)
	}, time.Second*2)

	if err != nil {
		logrus.WithError(err).Error("UDP Replay failed")
	}
}
