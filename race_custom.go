package servermanager

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type CustomRace struct {
	Name                            string
	HasCustomName, OverridePassword bool
	ReplacementPassword             string

	Created       time.Time
	Updated       time.Time
	Deleted       time.Time
	Scheduled     time.Time
	UUID          uuid.UUID
	Starred, Loop bool

	RaceConfig CurrentRaceConfig
	EntryList  EntryList
}

func (cr *CustomRace) EventName() string {
	if cr.HasCustomName {
		return cr.Name
	} else {
		return ""
	}
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

func (cr *CustomRace) HasSignUpForm() bool {
	return false
}

func (cr *CustomRace) GetID() uuid.UUID {
	return cr.UUID
}

func (cr *CustomRace) GetRaceSetup() CurrentRaceConfig {
	return cr.RaceConfig
}

func (cr *CustomRace) GetScheduledTime() time.Time {
	return cr.Scheduled
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

func (crh *CustomRaceHandler) list(w http.ResponseWriter, r *http.Request) {
	recent, starred, looped, scheduled, err := crh.raceManager.ListCustomRaces()

	if err != nil {
		logrus.Errorf("couldn't list custom races, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	crh.viewRenderer.MustLoadTemplate(w, r, "custom-race/index.html", map[string]interface{}{
		"Recent":    recent,
		"Starred":   starred,
		"Loop":      looped,
		"Scheduled": scheduled,
	})
}

func (crh *CustomRaceHandler) createOrEdit(w http.ResponseWriter, r *http.Request) {
	customRaceData, err := crh.raceManager.BuildRaceOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	crh.viewRenderer.MustLoadTemplate(w, r, "custom-race/new.html", customRaceData)
}

func (crh *CustomRaceHandler) submit(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.SetupCustomRace(r)

	if err != nil {
		logrus.Errorf("couldn't apply quick race, err: %s", err)
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
		http.Redirect(w, r, "/live-timing", http.StatusFound)
	}
}

func (crh *CustomRaceHandler) schedule(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		logrus.Errorf("couldn't parse schedule race form, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	raceID := chi.URLParam(r, "uuid")
	dateString := r.FormValue("event-schedule-date")
	timeString := r.FormValue("event-schedule-time")
	timezone := r.FormValue("event-schedule-timezone")

	location, err := time.LoadLocation(timezone)

	if err != nil {
		logrus.WithError(err).Errorf("could not find location: %s", location)
		location = time.Local
	}

	// Parse time in correct time zone
	date, err := time.ParseInLocation("2006-01-02-15:04", dateString+"-"+timeString, location)

	if err != nil {
		logrus.Errorf("couldn't parse schedule race date, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = crh.raceManager.ScheduleRace(raceID, date, r.FormValue("action"))

	if err != nil {
		logrus.Errorf("couldn't schedule race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, fmt.Sprintf("We have scheduled the race to begin at %s", date.Format(time.RFC1123)))
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (crh *CustomRaceHandler) removeSchedule(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.ScheduleRace(chi.URLParam(r, "uuid"), time.Time{}, "remove")

	if err != nil {
		logrus.Errorf("couldn't remove scheduled race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (crh *CustomRaceHandler) start(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.StartCustomRace(chi.URLParam(r, "uuid"), false)

	if err != nil {
		logrus.Errorf("couldn't apply custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Custom race started!")
	http.Redirect(w, r, "/live-timing", http.StatusFound)
}

func (crh *CustomRaceHandler) delete(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.DeleteCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.Errorf("couldn't delete custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Custom race deleted!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (crh *CustomRaceHandler) star(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.ToggleStarCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.Errorf("couldn't star custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (crh *CustomRaceHandler) loop(w http.ResponseWriter, r *http.Request) {
	err := crh.raceManager.ToggleLoopCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.Errorf("couldn't add custom race to loop, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
