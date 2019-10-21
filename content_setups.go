package servermanager

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cj123/ini"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

// CarSetups is a map of car name to the setups for that car.
// it is a map of car name => track name => []setup
type CarSetups map[string]map[string][]string

func ListAllSetups() (CarSetups, error) {
	setupDirectory := filepath.Join(ServerInstallPath, "setups")

	if _, err := os.Stat(setupDirectory); os.IsNotExist(err) {
		err := os.MkdirAll(setupDirectory, 0755)

		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	setups := make(CarSetups)

	// multi server installs may use symbolic links for setups folder, which filepath.Walk will ignore
	setupDirectory, err := filepath.EvalSymlinks(setupDirectory)

	if err != nil {
		setupDirectory = filepath.Join(ServerInstallPath, "setups")
	}

	err = filepath.Walk(setupDirectory, func(path string, file os.FileInfo, _ error) error {
		if file.IsDir() || filepath.Ext(file.Name()) != ".ini" {
			return nil
		}

		// corner case of stray ini files winding up in the top level of the folder
		if filepath.ToSlash(filepath.Dir(path)) == filepath.ToSlash(setupDirectory) {
			return nil
		}

		// read the setup file to get the car name
		name, err := getCarNameFromSetup(path)

		if err != nil {
			logrus.WithError(err).Errorf("Could not get car name from setup file %s", file.Name())
			return nil
		}

		if setups[name] == nil {
			setups[name] = make(map[string][]string)
		}

		trackParts := strings.Split(filepath.ToSlash(filepath.Dir(path)), "/")

		if len(trackParts) > 0 {
			trackName := trackParts[len(trackParts)-1]

			if trackName == lockedTyreSetupFolder {
				return nil // don't list locked tyre setup folder
			}

			setups[name][trackName] = append(setups[name][trackName], file.Name())
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return setups, nil
}

func ListSetupsForCar(model string) (map[string][]string, error) {
	setups := make(map[string][]string)

	setupDirectory := filepath.Join(ServerInstallPath, "setups", model)

	if _, err := os.Stat(setupDirectory); os.IsNotExist(err) {
		return setups, nil
	} else if err != nil {
		return nil, err
	}

	err := filepath.Walk(setupDirectory, func(path string, file os.FileInfo, _ error) error {
		if file.IsDir() || filepath.Ext(file.Name()) != ".ini" {
			return nil
		}

		trackParts := strings.Split(filepath.ToSlash(filepath.Dir(path)), "/")

		if len(trackParts) > 0 {
			trackName := trackParts[len(trackParts)-1]

			if trackName == lockedTyreSetupFolder {
				return nil // don't list locked tyre setup folder
			}

			setups[trackName] = append(setups[trackName], file.Name())
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return setups, nil
}

func getCarNameFromSetup(setupFile interface{}) (string, error) {
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

func uploadCarSetup(r *http.Request) (string, error) {
	track := r.FormValue("Track")

	uploadedFile, header, err := r.FormFile("SetupFile")

	if err != nil {
		return "", err
	}

	defer uploadedFile.Close()

	carName, err := getCarNameFromSetup(uploadedFile)

	if err != nil {
		return carName, err
	}

	_, err = uploadedFile.Seek(0, 0)

	if err != nil {
		return carName, err
	}

	saveFilepath := filepath.Join(ServerInstallPath, "setups", carName, track)

	if err := os.MkdirAll(saveFilepath, 0755); err != nil {
		return carName, err
	}

	savedFile, err := os.Create(filepath.Join(saveFilepath, header.Filename))

	if err != nil {
		return carName, err
	}

	defer savedFile.Close()

	_, err = io.Copy(savedFile, uploadedFile)

	return carName, err
}

func carSetupsUploadHandler(w http.ResponseWriter, r *http.Request) {
	if carName, err := uploadCarSetup(r); err != nil {
		logrus.WithError(err).Errorf("Could not upload setup file")

		if carName != "" {
			AddErrorFlash(w, r, fmt.Sprintf("Unable to upload setup file for %s", carName))
		} else {
			AddErrorFlash(w, r, "Unable to upload setup file")
		}
	} else {
		AddFlash(w, r, fmt.Sprintf("The setup file for %s was uploaded successfully!", carName))
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func carSetupDeleteHandler(w http.ResponseWriter, r *http.Request) {
	carName := chi.URLParam(r, "car")
	trackName := chi.URLParam(r, "track")
	setupName := chi.URLParam(r, "setup")

	err := os.RemoveAll(filepath.Join(ServerInstallPath, "setups", carName, trackName, setupName))

	if err != nil {
		logrus.WithError(err).Errorf("Could not remove setup %s/%s/%s", carName, trackName, setupName)
		AddErrorFlash(w, r, "Couldn't delete setup for "+carName)
	} else {
		AddFlash(w, r, "Setup successfully deleted!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
