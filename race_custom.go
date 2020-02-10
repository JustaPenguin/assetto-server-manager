package servermanager

import (
	"fmt"
	"net/http"
	"time"

	"4d63.com/tz"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type CustomRace struct {
	ScheduledEventBase

	Name                            string
	HasCustomName, OverridePassword bool
	ReplacementPassword             string

	Created time.Time
	Updated time.Time
	Deleted time.Time
	UUID    uuid.UUID
	Starred bool
	// Deprecated: Replaced by LoopServer
	Loop       bool
	LoopServer map[ServerID]bool

	ForceStopTime        int
	ForceStopWithDrivers bool

	RaceConfig CurrentRaceConfig
	EntryList  EntryList
}

func (cr *CustomRace) GetRaceConfig() CurrentRaceConfig {
	return cr.RaceConfig
}

func (cr *CustomRace) GetEntryList() EntryList {
	return cr.EntryList
}

func (cr *CustomRace) IsLooping() bool {
	if cr.LoopServer == nil {
		return false
	}

	return cr.LoopServer[serverID]
}

func (cr *CustomRace) EventName() string {
	if cr.HasCustomName {
		return cr.Name
	}

	return trackSummary(cr.RaceConfig.Track, cr.RaceConfig.TrackLayout)
}

func (cr *CustomRace) OverrideServerPassword() bool {
	return cr.OverridePassword
}

func (cr *CustomRace) ReplacementServerPassword() string {
	return cr.ReplacementPassword
}

func (cr *CustomRace) IsChampionship() bool {
	return false
}

func (cr *CustomRace) IsPractice() bool {
	return false
}

func (cr *CustomRace) IsRaceWeekend() bool {
	return false
}

func (cr *CustomRace) HasSignUpForm() bool {
	return false
}

func (cr *CustomRace) GetID() uuid.UUID {
	return cr.UUID
}

func (cr *CustomRace) GetRaceSetup() CurrentRaceConfig {
	return cr.RaceConfig
}

func (cr *CustomRace) GetSummary() string {
	return ""
}

func (cr *CustomRace) GetURL() string {
	return ""
}

func (cr *CustomRace) EventDescription() string {
	return ""
}

func (cr *CustomRace) ReadOnlyEntryList() EntryList {
	return cr.EntryList
}

func (cr *CustomRace) GetForceStopTime() time.Duration {
	return time.Minute * time.Duration(cr.ForceStopTime)
}

func (cr *CustomRace) GetForceStopWithDrivers() bool {
	return cr.ForceStopWithDrivers
}

type CustomRaceHandler struct {
	*BaseHandler

	raceManager *RaceManager
}

func NewCustomRaceHandler(base *BaseHandler, raceManager *RaceManager) *CustomRaceHandler {
	return &CustomRaceHandler{
		BaseHandler: base,
		raceManager: raceManager,
	}
}

type customRaceListTemplateVars struct {
	BaseTemplateVars

	Recent, Starred, Loop, Scheduled []*CustomRace
}

func (crh *CustomRaceHandler) list(w http.ResponseWriter, r *http.Request) {
	recent, starred, looped, scheduled, err := crh.raceManager.ListCustomRaces()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't list custom races")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	crh.viewRenderer.MustLoadTemplate(w, r, "custom-race/index.html", &customRaceListTemplateVars{
		Recent:    recent,
		Starred:   starred,
		Loop:      looped,
		Scheduled: scheduled,
	})
}

func (crh *CustomRaceHandler) createOrEdit(w http.ResponseWriter, r *http.Request) {
	customRaceData, err := crh.raceManager.BuildRaceOpts(r)

	if err != nil {
		logrus.WithError(err).Errorf("couldn't build custom race")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	crh.viewRenderer.MustLoadTemplate(w, r, "custom-race/new.html", customRaceData)
}

func (crh *CustomRaceHandler) submit(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.SetupCustomRace(r)

	if err != nil {
		logrus.WithError(err).Errorf("couldn't apply quick race")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	action := r.FormValue("action")

	if action == "justSave" {
		AddFlash(w, r, "Custom race saved!")
		http.Redirect(w, r, "/custom", http.StatusFound)
	} else if action == "schedule" {
		AddFlash(w, r, "Custom race scheduled!")
		http.Redirect(w, r, "/custom", http.StatusFound)
	} else {
		AddFlash(w, r, "Custom race started!")
		if config.Server.PerformanceMode {
			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			http.Redirect(w, r, "/live-timing", http.StatusFound)
		}
	}
}

func (crh *CustomRaceHandler) schedule(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		logrus.WithError(err).Errorf("couldn't parse schedule race form")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	raceID := chi.URLParam(r, "uuid")
	dateString := r.FormValue("event-schedule-date")
	timeString := r.FormValue("event-schedule-time")
	timezone := r.FormValue("event-schedule-timezone")

	location, err := tz.LoadLocation(timezone)

	if err != nil {
		logrus.WithError(err).Errorf("could not find location: %s", location)
		location = time.Local
	}

	// Parse time in correct time zone
	date, err := time.ParseInLocation("2006-01-02-15:04", dateString+"-"+timeString, location)

	if err != nil {
		logrus.WithError(err).Errorf("couldn't parse schedule race date")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = crh.raceManager.ScheduleRace(raceID, date, r.FormValue("action"), r.FormValue("event-schedule-recurrence"))

	if err != nil {
		logrus.WithError(err).Errorf("couldn't schedule race")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, fmt.Sprintf("We have scheduled the race to begin at %s", date.Format(time.RFC1123)))
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (crh *CustomRaceHandler) removeSchedule(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.ScheduleRace(chi.URLParam(r, "uuid"), time.Time{}, "remove", "")

	if err != nil {
		logrus.WithError(err).Errorf("couldn't remove scheduled race")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (crh *CustomRaceHandler) start(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.StartCustomRace(chi.URLParam(r, "uuid"), false)

	if err != nil {
		logrus.WithError(err).Errorf("couldn't apply custom race")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Custom race started!")

	if config.Server.PerformanceMode {
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		http.Redirect(w, r, "/live-timing", http.StatusFound)
	}
}

func (crh *CustomRaceHandler) delete(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.DeleteCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.WithError(err).Errorf("couldn't delete custom race")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Custom race deleted!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (crh *CustomRaceHandler) star(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.ToggleStarCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.WithError(err).Errorf("couldn't star custom race")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (crh *CustomRaceHandler) loop(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.ToggleLoopCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.WithError(err).Errorf("couldn't add custom race to loop")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
