package servermanager

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

type QuickRaceHandler struct {
	*BaseHandler

	raceManager *RaceManager
}

func NewQuickRaceHandler(baseHandler *BaseHandler, raceManager *RaceManager) *QuickRaceHandler {
	return &QuickRaceHandler{
		BaseHandler: baseHandler,
		raceManager: raceManager,
	}
}

func (qrh *QuickRaceHandler) create(w http.ResponseWriter, r *http.Request) {
	quickRaceData, err := qrh.raceManager.BuildRaceOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build quick race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	qrh.viewRenderer.MustLoadTemplate(w, r, "quick-race.html", quickRaceData)
}

func (qrh *QuickRaceHandler) submit(w http.ResponseWriter, r *http.Request) {
	err := qrh.raceManager.SetupQuickRace(r)

	if err == ErrMustSubmitCar {
		AddErrorFlash(w, r, "You must choose at least one car!")
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	} else if err != nil {
		logrus.Errorf("couldn't apply quick race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Quick race successfully started!")
	http.Redirect(w, r, "/live-timing", http.StatusFound)
}
