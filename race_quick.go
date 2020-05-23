package servermanager

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

func (ms *MultiServer) quickRaceHandler(w http.ResponseWriter, r *http.Request) {
	quickRaceData, err := ms.raceManager.BuildRaceOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build quick race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "quick-race.html", quickRaceData)
}

func (ms *MultiServer) quickRaceSubmitHandler(w http.ResponseWriter, r *http.Request) {
	err := ms.raceManager.SetupQuickRace(r)

	if err == ErrMustSubmitCar {
		AddErrFlashQuick(w, r, "You must choose at least one car!")
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	} else if err != nil {
		logrus.Errorf("couldn't apply quick race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlashQuick(w, r, "Quick race successfully started!")
	http.Redirect(w, r, "/", http.StatusFound)
}
