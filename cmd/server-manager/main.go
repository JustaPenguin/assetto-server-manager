package main

import (
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/etcd-io/bbolt"

	"github.com/cj123/assetto-server-manager"
	"github.com/cj123/assetto-server-manager/cmd/server-manager/static"
	"github.com/cj123/assetto-server-manager/cmd/server-manager/views"
	"github.com/cj123/assetto-server-manager/pkg/udp"
	"github.com/cj123/assetto-server-manager/pkg/udp/replay"

	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
)

var debug = os.Getenv("DEBUG") == "true"

var defaultAddress = "0.0.0.0:8772"

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
		if config.LiveMap.IntervalMs < 200 {
			udp.RealtimePosIntervalMs = 200
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

	//go startUDPReplay("./assetto/session-logs/brandshatch_sillyold.db")
	//go MiniRace()

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
	}, time.Second*2)

	if err != nil {
		logrus.WithError(err).Error("UDP Replay failed")
	}
}

func MiniRace() {
	time.Sleep(time.Second * 5)

	do := servermanager.ServerRaceControl.UDPCallback

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
		WeatherGraphics:     "3_clear",
		ElapsedMilliseconds: 10,

		EventType: udp.EventNewSession,
	})
	do(udp.SessionCarInfo{
		CarID:      1,
		DriverName: "Test 1",
		DriverGUID: "7827162738272615",
		CarModel:   "ford_gt",
		CarSkin:    "red_01",
		EventType:  udp.EventNewConnection,
	})

	time.Sleep(time.Second * 2)

	do(udp.ClientLoaded(1))

	for i := 0; i < 10; i++ {
		time.Sleep(time.Second * 3)

		do(udp.LapCompleted{
			CarID:     1,
			LapTime:   uint32(rand.Intn(10000)),
			Cuts:      0,
			CarsCount: 1,
		})
	}

	do(udp.SessionCarInfo{
		CarID:      1,
		DriverName: "Test 1",
		DriverGUID: "7827162738272615",
		CarModel:   "ford_gt",
		CarSkin:    "red_01",
		EventType:  udp.EventConnectionClosed,
	})

	time.Sleep(2 * time.Second)

	do(udp.SessionCarInfo{
		CarID:      1,
		DriverName: "Test 1",
		DriverGUID: "7827162738272615",
		CarModel:   "ferrari_fxx_k",
		CarSkin:    "red_01",
		EventType:  udp.EventNewConnection,
	})

	do(udp.ClientLoaded(1))

	for i := 0; i < 6; i++ {
		time.Sleep(time.Second * 3)

		do(udp.LapCompleted{
			CarID:     1,
			LapTime:   uint32(rand.Intn(1000000)),
			Cuts:      0,
			CarsCount: 1,
		})
	}

	do(udp.SessionCarInfo{
		CarID:      1,
		DriverName: "Test 1",
		DriverGUID: "7827162738272615",
		CarModel:   "ferrari_fxx_k",
		CarSkin:    "red_01",
		EventType:  udp.EventConnectionClosed,
	})

	time.Sleep(2 * time.Second)

	do(udp.SessionCarInfo{
		CarID:      4,
		DriverName: "Test 1",
		DriverGUID: "7827162738272615",
		CarModel:   "ferrari_fxx_k",
		CarSkin:    "red_01",
		EventType:  udp.EventNewConnection,
	})

	do(udp.SessionCarInfo{
		CarID:      30,
		DriverName: "Test 2",
		DriverGUID: "7827162738272677",
		CarModel:   "ferrari_fxx_k",
		CarSkin:    "red_01",
		EventType:  udp.EventNewConnection,
	})

	do(udp.ClientLoaded(1))
	do(udp.ClientLoaded(30))

	for i := 0; i < 20; i++ {
		time.Sleep(time.Second * 3)

		do(udp.LapCompleted{
			CarID:     4,
			LapTime:   uint32(rand.Intn(10000)),
			Cuts:      0,
			CarsCount: 1,
		})

		do(udp.LapCompleted{
			CarID:     30,
			LapTime:   uint32(rand.Intn(1000000)),
			Cuts:      0,
			CarsCount: 1,
		})
	}

	do(udp.SessionCarInfo{
		CarID:      4,
		DriverName: "Test 1",
		DriverGUID: "7827162738272615",
		CarModel:   "ferrari_fxx_k",
		CarSkin:    "red_01",
		EventType:  udp.EventConnectionClosed,
	})

	do(udp.SessionCarInfo{
		CarID:      30,
		DriverName: "Test 2",
		DriverGUID: "7827162738272677",
		CarModel:   "ferrari_fxx_k",
		CarSkin:    "red_01",
		EventType:  udp.EventConnectionClosed,
	})

	do(udp.SessionCarInfo{
		CarID:      1,
		DriverName: "Test 1",
		DriverGUID: "7827162738272615",
		CarModel:   "ford_gt",
		CarSkin:    "red_01",
		EventType:  udp.EventNewConnection,
	})

	time.Sleep(time.Second * 2)

	do(udp.ClientLoaded(1))

	for i := 0; i < 2; i++ {
		time.Sleep(time.Second * 3)

		do(udp.LapCompleted{
			CarID:     1,
			LapTime:   uint32(rand.Intn(10000)),
			Cuts:      0,
			CarsCount: 1,
		})
	}

	do(udp.SessionCarInfo{
		CarID:      1,
		DriverName: "Test 1",
		DriverGUID: "7827162738272615",
		CarModel:   "ford_gt",
		CarSkin:    "red_01",
		EventType:  udp.EventConnectionClosed,
	})
}
