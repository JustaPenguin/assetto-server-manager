package servermanager

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

func quickRaceHandler(w http.ResponseWriter, r *http.Request) {
	quickRaceData, err := raceManager.BuildRaceOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build quick race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "quick_race.html", quickRaceData)
}

func quickRaceSubmitHandler(w http.ResponseWriter, r *http.Request) {
	err := raceManager.SetupQuickRace(r)

	if err != nil {
		logrus.Errorf("couldn't apply quick race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlashQuick(w, r, "Quick race successfully started!")
	http.Redirect(w, r, "/", http.StatusFound)
}
