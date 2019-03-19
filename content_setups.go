package servermanager

import (
	"github.com/cj123/ini"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

// CarSetups is a map of car name to the setups for that car.
type CarSetups map[string][]string

func ListSetups() (CarSetups, error) {

	setups := make(CarSetups)

	err := filepath.Walk(filepath.Join(ServerInstallPath, "setups"), func(path string, file os.FileInfo, err error) error {
		if file.IsDir() || filepath.Ext(file.Name()) != ".ini" {
			return nil
		}

		// read the setup file to get the car name
		name, err := getCarNameFromSetup(path)

		if err != nil {
			logrus.Errorf("Could not get car name from setup file %s, err: %s", file.Name(), err)
			return nil
		}

		setups[name] = append(setups[name], file.Name())

		return nil
	})

	if err != nil {
		return nil, err
	}

	return setups, nil
}

func getCarNameFromSetup(setupFile string) (string, error) {
	i, err := ini.Load(setupFile)

	if err != nil {
		return "", err
	}

	car, err := i.GetSection("CAR")

	if err != nil {
		return "", err
	}

	name, err := car.GetKey("MODEL")

	if err != nil {
		return "", err
	}

	return name.String(), nil
}

func carSetupsHandler(w http.ResponseWriter, r *http.Request) {
	setups, err := ListSetups()

	if err != nil {
		logrus.Errorf("could not get track list, err: %s", err)
	}

	ViewRenderer.MustLoadTemplate(w, r, "content/setups.html", map[string]interface{}{
		"setups": setups,
	})
}

func apiCarSetupsUploadHandler(w http.ResponseWriter, r *http.Request) {
	uploadHandler(w, r, "Track")
}

func carSetupDeleteHandler(w http.ResponseWriter, r *http.Request) {
	trackName := chi.URLParam(r, "name")
	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")

	existingTracks, err := ListTracks()

	if err != nil {
		logrus.Errorf("could not get track list, err: %s", err)

		AddErrFlashQuick(w, r, "couldn't get track list")

		http.Redirect(w, r, r.Referer(), http.StatusFound)

		return
	}

	var found bool

	for _, track := range existingTracks {
		if track.Name == trackName {
			// Delete track
			found = true

			err := os.RemoveAll(filepath.Join(tracksPath, trackName))

			if err != nil {
				found = false
				logrus.Errorf("could not remove track files, err: %s", err)
			}

			break
		}
	}

	if found {
		// confirm deletion
		AddFlashQuick(w, r, "Track successfully deleted!")
	} else {
		// inform track wasn't found
		AddErrFlashQuick(w, r, "Sorry, track could not be deleted.")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
