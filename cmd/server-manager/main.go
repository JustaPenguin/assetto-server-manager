package main

import (
	"net/http"
	"os"

	"github.com/cj123/assetto-server-manager"
	"github.com/cj123/assetto-server-manager/cmd/server-manager/static"
	"github.com/cj123/assetto-server-manager/cmd/server-manager/views"

	"github.com/sirupsen/logrus"
)

var debug = os.Getenv("DEBUG") == "true"

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

	var templateLoader servermanager.TemplateLoader
	var filesystem http.FileSystem

	if debug {
		templateLoader = servermanager.NewFilesystemTemplateLoader("views")
		filesystem = http.Dir("static")
	} else {
		templateLoader = &views.TemplateLoader{}
		filesystem = static.FS(false)
	}

	servermanager.ViewRenderer, err = servermanager.NewRenderer(templateLoader, debug)

	if err != nil {
		logrus.Fatalf("could not initialise view renderer, err: %s", err)
	}

	logrus.Infof("starting assetto server manager on: %s", config.HTTP.Hostname)
	logrus.Fatal(http.ListenAndServe(config.HTTP.Hostname, servermanager.Router(filesystem)))
}
