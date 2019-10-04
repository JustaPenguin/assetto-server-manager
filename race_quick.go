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

	if config.Server.PerformanceMode {
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		http.Redirect(w, r, "/live-timing", http.StatusFound)
	}
}

type QuickRace struct {
	OverridePassword    bool
	ReplacementPassword string
	RaceConfig          CurrentRaceConfig
}

func (q QuickRace) IsChampionship() bool {
	return false
}

func (q QuickRace) EventName() string {
	return trackSummary(q.RaceConfig.Track, q.RaceConfig.TrackLayout)
}

func (q QuickRace) OverrideServerPassword() bool {
	return q.OverridePassword
}

func (q QuickRace) ReplacementServerPassword() string {
	return q.ReplacementPassword
}

func (q QuickRace) EventDescription() string {
	return ""
}

func (q QuickRace) GetURL() string {
	return ""
}
