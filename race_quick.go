package servermanager

import (
	"net/http"
	"time"

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
		logrus.WithError(err).Errorf("couldn't build quick race")
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
		logrus.WithError(err).Errorf("couldn't apply quick race")
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
	EntryList           EntryList
}

func (q QuickRace) GetRaceConfig() CurrentRaceConfig {
	return q.RaceConfig
}

func (q QuickRace) GetEntryList() EntryList {
	return q.EntryList
}

func (q QuickRace) IsLooping() bool {
	return false
}

func (q QuickRace) IsPractice() bool {
	return false
}

func (q QuickRace) IsChampionship() bool {
	return false
}

func (q QuickRace) IsRaceWeekend() bool {
	return false
}

func (q QuickRace) IsTimeAttack() bool {
	return q.RaceConfig.TimeAttack
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

func (q QuickRace) GetForceStopTime() time.Duration {
	return 0
}

func (q QuickRace) GetForceStopWithDrivers() bool {
	return false
}
