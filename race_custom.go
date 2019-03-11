package servermanager

import (
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type CustomRace struct {
	Name          string
	Created       time.Time
	Updated       time.Time
	Deleted       time.Time
	Scheduled     time.Time
	UUID          uuid.UUID
	Starred, Loop bool

	RaceConfig CurrentRaceConfig
	EntryList  EntryList
}

func customRaceListHandler(w http.ResponseWriter, r *http.Request) {
	recent, starred, looped, scheduled, err := raceManager.ListCustomRaces()

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

func customRaceNewOrEditHandler(w http.ResponseWriter, r *http.Request) {
	customRaceData, err := raceManager.BuildRaceOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "custom-race/new.html", customRaceData)
}

func customRaceSubmitHandler(w http.ResponseWriter, r *http.Request) {
	err := raceManager.SetupCustomRace(r)

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
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func customRaceScheduleHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		logrus.Errorf("couldn't parse schedule race form, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	raceID := chi.URLParam(r, "uuid")
	dateString := r.FormValue("event-schedule-date")
	timeString := r.FormValue("event-schedule-time")

	dateTimeString := dateString + "-" + timeString

	date, err := time.Parse("2006-01-02-15:04", dateTimeString)

	if err != nil {
		logrus.Errorf("couldn't parse schedule race date, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = raceManager.ScheduleRace(raceID, date, r.FormValue("action"))

	if err != nil {
		logrus.Errorf("couldn't schedule race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func customRaceLoadHandler(w http.ResponseWriter, r *http.Request) {
	err := raceManager.StartCustomRace(chi.URLParam(r, "uuid"), false)

	if err != nil {
		logrus.Errorf("couldn't apply custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlashQuick(w, r, "Custom race started!")
	http.Redirect(w, r, "/", http.StatusFound)
}

func customRaceDeleteHandler(w http.ResponseWriter, r *http.Request) {
	err := raceManager.DeleteCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.Errorf("couldn't delete custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlashQuick(w, r, "Custom race deleted!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func customRaceStarHandler(w http.ResponseWriter, r *http.Request) {
	err := raceManager.ToggleStarCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.Errorf("couldn't star custom race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func customRaceLoopHandler(w http.ResponseWriter, r *http.Request) {
	err := raceManager.ToggleLoopCustomRace(chi.URLParam(r, "uuid"))

	if err != nil {
		logrus.Errorf("couldn't add custom race to loop, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
