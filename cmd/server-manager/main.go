package main

import (
	"net/http"

	"github.com/cj123/assetto-server-manager"

	"github.com/sirupsen/logrus"
)

func main() {
	config, err := servermanager.ReadConfig("config.yml")

	if err != nil {
		logrus.Fatalf("could not open config file, err: %s", err)
	}

	store, err := config.Store.BuildStore()

	if err != nil {
		logrus.Fatalf("could not open store, err: %s", err)
	}

	servermanager.SetupRaceManager(store)
	servermanager.SetAssettoInstallPath(config.Steam.InstallPath)

	err = servermanager.InstallAssettoCorsaServer(config.Steam.Username, config.Steam.Password, config.Steam.ForceUpdate)

	if err != nil {
		logrus.Fatalf("could not install assetto corsa server, err: %s", err)
	}

	servermanager.ViewRenderer, err = servermanager.NewRenderer("./views", true)

	if err != nil {
		logrus.Fatalf("could not initialise view renderer, err: %s", err)
	}

	go servermanager.LoopRaces()

	logrus.Infof("starting assetto server manager on: %s", config.HTTP.Hostname)
	logrus.Fatal(http.ListenAndServe(config.HTTP.Hostname, servermanager.Router()))
}
