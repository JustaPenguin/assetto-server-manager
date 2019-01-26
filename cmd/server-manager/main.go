package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cj123/assetto-server-manager"
	"github.com/sirupsen/logrus"
)

var (
	steamUsername = os.Getenv("STEAM_USERNAME")
	steamPassword = os.Getenv("STEAM_PASSWORD")
)

func main() {
	err := servermanager.InstallAssettoCorsaServer(steamUsername, steamPassword, false)

	if err != nil {
		logrus.Fatalf("could not install assetto corsa server, err: %s", err)
	}

	serverProcess := servermanager.AssettoServerProcess{}
	err = serverProcess.Start()

	if err != nil {
		logrus.Fatal(err)
	}

	time.Sleep(time.Second * 10)

	fmt.Println(serverProcess.Logs())

	logrus.Info(serverProcess.Status())

	err = serverProcess.Stop()
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info(serverProcess.Status())
	fmt.Println(serverProcess.Logs())

	err = serverProcess.Stop()
	if err != nil {
		logrus.Fatal(err)
	}
}
