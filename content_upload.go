package servermanager

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

type ContentFile struct {
	Name     string `json:"name"`
	FileType string `json:"type"`
	FilePath string `json:"filepath"`
	Data     string `json:"dataBase64"`
	Size     int    `json:"size"`
}

var base64HeaderRegex = regexp.MustCompile("^(data:.+;base64,)")

// Stores Files encoded into r.Body
func uploadHandler(contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var files []ContentFile

		decoder := json.NewDecoder(r.Body)

		err := decoder.Decode(&files)

		if err != nil {
			logrus.Errorf("could not decode "+strings.ToLower(contentType)+" json, err: %s", err)
			return
		}

		err = addFiles(files, contentType)

		if err != nil {
			AddErrorFlash(w, r, contentType+"(s) could not be added")

			return
		}

		AddFlash(w, r, contentType+"(s) added successfully!")
	}
}

// Stores files in the correct location
func addFiles(files []ContentFile, contentType string) error {
	var contentPath string

	switch contentType {
	case "Track":
		contentPath = filepath.Join(ServerInstallPath, "content", "tracks")
	case "Car":
		contentPath = filepath.Join(ServerInstallPath, "content", "cars")
	case "Weather":
		contentPath = filepath.Join(ServerInstallPath, "content", "weather")
	}

	for _, file := range files {
		var fileDecoded []byte

		if file.Size > 0 {
			// zero-size files will still be created, just with no content. (some data files exist but are empty)
			var err error
			fileDecoded, err = base64.StdEncoding.DecodeString(base64HeaderRegex.ReplaceAllString(file.Data, ""))

			if err != nil {
				logrus.Errorf("could not decode "+contentType+" file data, err: %s", err)
				return err
			}
		}

		// If user uploaded a "tracks" or "cars" folder containing multiple tracks
		parts := strings.Split(file.FilePath, "/")

		if parts[0] == "tracks" || parts[0] == "cars" || parts[0] == "weather" {
			parts = parts[1:]
			file.FilePath = ""

			for _, part := range parts {
				file.FilePath = filepath.Join(file.FilePath, part)
			}
		}

		path := filepath.Join(contentPath, file.FilePath)

		// Makes any directories in the path that don't exist (there can be multiple)
		err := os.MkdirAll(filepath.Dir(path), 0755)

		if err != nil {
			logrus.Errorf("could not create "+contentType+" file directory, err: %s", err)
			return err
		}

		if contentType == "Car" {
			if _, name := filepath.Split(file.FilePath); name == "data.acd" {
				err := addTyresFromDataACD(file.FilePath, fileDecoded)

				if err != nil {
					logrus.Errorf("Could not create tyres for new car (%s), err: %s", file.FilePath, err)
				}
			} else if name == "tyres.ini" {
				// it seems some cars don't pack their data into an ACD file, it's just in a folder called 'data'
				// so we can just grab tyres.ini from there.
				err := addTyresFromTyresIni(file.FilePath, fileDecoded)

				if err != nil {
					logrus.Errorf("Could not create tyres for new car (%s), err: %s", file.FilePath, err)
				}
			}
		}

		err = ioutil.WriteFile(path, fileDecoded, 0644)

		if err != nil {
			logrus.Errorf("could not write file, err: %s", err)
			return err
		}
	}

	return nil
}
