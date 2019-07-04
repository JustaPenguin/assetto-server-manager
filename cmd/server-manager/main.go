package main

import (
	"math/rand"
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

var (
	defaultAddress = "0.0.0.0:8772"
)

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
		if config.LiveMap.IntervalMs < udpRealtimePosRefreshIntervalMin {
			udp.RealtimePosIntervalMs = udpRealtimePosRefreshIntervalMin
		} else {
			udp.RealtimePosIntervalMs = config.LiveMap.IntervalMs
		}
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

	go startUDPReplay("./assetto/session-logs/2019-04-05_13.27.db")
	//go TestRace()

	listener, err := net.Listen("tcp", config.HTTP.Hostname)

	if err != nil {
		ServeHTTPWithError(defaultAddress, "Listen on hostname "+config.HTTP.Hostname+". Likely the port has already been taken by another application", err)
		return
	}

	logrus.Infof("starting assetto server manager on: %s", config.HTTP.Hostname)

	if runtime.GOOS == "windows" {
		_ = browser.OpenURL("http://" + strings.Replace(config.HTTP.Hostname, "0.0.0.0", "127.0.0.1", 1))
	}

	router := servermanager.Router(filesystem)

	if err := http.Serve(listener, router); err != nil {
		logrus.Fatal(err)
	}
}

func startUDPReplay(file string) {
	time.Sleep(time.Second * 20)

	db, err := bbolt.Open(file, 0644, nil)

	if err != nil {
		logrus.WithError(err).Error("Could not open bolt")
	}

	err = replay.ReplayUDPMessages(db, 1, func(response udp.Message) {
		servermanager.ServerRaceControl.UDPCallback(response)
	}, time.Millisecond*500)

	if err != nil {
		logrus.WithError(err).Error("UDP Replay failed")
	}
}

func TestRace() {
	time.Sleep(time.Second * 10)

	do := servermanager.ServerRaceControl.UDPCallback

	rand.Seed(time.Now().Unix())

	do(udp.Version(4))
	do(udp.SessionInfo{
		Version:             4,
		SessionIndex:        0,
		CurrentSessionIndex: 0,
		SessionCount:        3,
		ServerName:          "Test Server",
		Track:               "ks_laguna_seca",
		TrackConfig:         "",
		Name:                "Test Practice Session",
		Type:                udp.SessionTypePractice,
		Time:                10,
		Laps:                0,
		WaitTime:            120,
		AmbientTemp:         12,
		RoadTemp:            16,
		WeatherGraphics:     "01_clear",
		ElapsedMilliseconds: 10,

		EventType: udp.EventNewSession,
	})

	for _, car := range []string{"ks_lamborghini_huracan", "bmw_m3", "ferrari_fxx_k", "ks_lamborghini_huracan"} {
		id := udp.CarID(rand.Intn(30))

		do(udp.SessionCarInfo{
			CarID:      id,
			DriverName: "Callum",
			DriverGUID: "78273827382738273",
			CarModel:   car,
			CarSkin:    "purple",
			EventType:  udp.EventNewConnection,
		})

		do(udp.ClientLoaded(id))

		for i := 0; i < rand.Intn(140)+10; i++ {
			do(udp.LapCompleted{
				CarID:   id,
				LapTime: 1000000+uint32(rand.Intn(300000)),
				Cuts:    0,
			})
			time.Sleep(time.Second)
		}

		do(udp.SessionCarInfo{
			CarID:      id,
			DriverName: "Callum",
			DriverGUID: "78273827382738273",
			CarModel:   car,
			CarSkin:    "purple",
			EventType:  udp.EventConnectionClosed,
		})

		time.Sleep(time.Second * 5)
	}
}
