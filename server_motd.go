package servermanager

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const MOTDFilename = "motd.txt"

func serverMOTDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		text := r.FormValue("motd")

		err := ioutil.WriteFile(filepath.Join(ServerInstallPath, MOTDFilename), []byte(text), 0644)

		if err != nil {
			logrus.WithError(err).Error("couldn't save message of the day")
			AddErrFlashQuick(w, r, "Failed to save message of the day changes")
		} else {
			AddFlashQuick(w, r, "Server message of the day successfully changed!")
		}
	}

	b, err := ioutil.ReadFile(filepath.Join(ServerInstallPath, MOTDFilename))

	if err != nil && !os.IsNotExist(err) {
		logrus.WithError(err).Error("couldn't find motd.txt")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "server/motd.html", map[string]interface{}{
		"text": string(b),
	})
}
