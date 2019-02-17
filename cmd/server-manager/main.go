package main

import (
	"net/http"
	"os"

	"github.com/cj123/assetto-server-manager"
	"github.com/etcd-io/bbolt"
	"github.com/sirupsen/logrus"
)

var (
	steamUsername = os.Getenv("STEAM_USERNAME")
	steamPassword = os.Getenv("STEAM_PASSWORD")

	serverAddress = os.Getenv("SERVER_ADDRESS")

	storeLocation = os.Getenv("STORE_LOCATION")
)

func main() {
	bboltdb, err := bbolt.Open(storeLocation, 0644, nil)

	if err != nil {
		logrus.Fatalf("could not open bbolt store at: '%s', err: %s", storeLocation, err)
	}

	servermanager.SetupRaceManager(servermanager.NewBoltRaceStore(bboltdb))

	err = servermanager.InstallAssettoCorsaServer(steamUsername, steamPassword, os.Getenv("FORCE_UPDATE") == "true")

	if err != nil {
		logrus.Fatalf("could not install assetto corsa server, err: %s", err)
	}

	servermanager.ViewRenderer, err = servermanager.NewRenderer("./views", true)

	if err != nil {
		logrus.Fatalf("could not initialise view renderer, err: %s", err)
	}

	logrus.Infof("starting assetto server manager on: %s", serverAddress)
	logrus.Fatal(http.ListenAndServe(serverAddress, servermanager.Router()))
}
