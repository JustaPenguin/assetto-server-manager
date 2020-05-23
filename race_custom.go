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

func (cr *CustomRace) GetEntryList() EntryList {
	return cr.EntryList
}

func (ms *MultiServer) customRaceListHandler(w http.ResponseWriter, r *http.Request) {
	recent, starred, looped, scheduled, err := ms.raceManager.ListCustomRaces()

	if err != nil {
		logrus.Errorf("couldn't list custom races, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "custom-race/index.html", map[string]interface{}{
		"Recent":    recent,
		"Starred":   starred,
		"Loop":      looped,
		"Scheduled": scheduled,
	})
}

func (ms *MultiServer) customRaceNewOrEditHandler(w http.ResponseWriter, r *http.Request) {
	customRaceData, err := ms.raceManager.BuildRaceOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "custom-race/new.html", customRaceData)
}

func (ms *MultiServer) customRaceSubmitHandler(w http.ResponseWriter, r *http.Request) {
	err := ms.raceManager.SetupCustomRace(r, ms.raceScheduler)

	if err != nil {
		logrus.Errorf("couldn't apply quick race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	action := r.FormValue("action")

	if action == "justSave" {
		AddFlashQuick(w, r, "Custom race saved!")
		http.Redirect(w, r, "/custom", http.StatusFound)
	} else if action == "schedule" {
		AddFlashQuick(w, r, "Custom race scheduled!")
		http.Redirect(w, r, "/custom", http.StatusFound)
	} else {
		AddFlashQuick(w, r, "Custom race started!")
		http.Redirect(w, r, "/live-timing", http.StatusFound)
	}
}

func (ms *MultiServer) customRaceScheduleHandler(w http.ResponseWriter, r *http.Request) {
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

	err = ms.raceScheduler.ScheduleRace(raceID, date, r.FormValue("action"))

	if err != nil {
		logrus.Errorf("couldn't schedule race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlashQuick(w, r, fmt.Sprintf("We have scheduled the race to begin at %s", date.Format(time.RFC1123)))
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ms *MultiServer) customRaceScheduleRemoveHandler(w http.ResponseWriter, r *http.Request) {
	err := ms.raceScheduler.ScheduleRace(chi.URLParam(r, "uuid"), time.Time{}, "remove")

	if err != nil {
		logrus.Errorf("couldn't remove scheduled race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ms *MultiServer) customRaceLoadHandler(w http.ResponseWriter, r *http.Request) {
	err := ms.raceManager.StartCustomRace(chi.URLParam(r, "uuid"), false)

	if err != nil {
		logrus.Errorf("couldn't apply custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlashQuick(w, r, "Custom race started!")
	http.Redirect(w, r, "/", http.StatusFound)
}

func (ms *MultiServer) customRaceDeleteHandler(w http.ResponseWriter, r *http.Request) {
	err := ms.raceManager.DeleteCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.Errorf("couldn't delete custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlashQuick(w, r, "Custom race deleted!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ms *MultiServer) customRaceStarHandler(w http.ResponseWriter, r *http.Request) {
	err := ms.raceManager.ToggleStarCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.Errorf("couldn't star custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ms *MultiServer) customRaceLoopHandler(w http.ResponseWriter, r *http.Request) {
	err := ms.raceManager.ToggleLoopCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.Errorf("couldn't add custom race to loop, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
