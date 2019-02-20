package servermanager

import (
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type CustomRace struct {
	Name    string
	Created time.Time
	Updated time.Time
	Deleted time.Time
	UUID    uuid.UUID
	Starred bool

	RaceConfig CurrentRaceConfig
	EntryList  EntryList
}

func customRaceListHandler(w http.ResponseWriter, r *http.Request) {
	recent, starred, err := raceManager.ListCustomRaces()

	if err != nil {
		logrus.Errorf("couldn't list custom races, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "custom-race/index.html", map[string]interface{}{
		"Recent":  recent,
		"Starred": starred,
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

	if r.FormValue("action") == "justSave" {
		AddFlashQuick(w, r, "Custom race saved!")
		http.Redirect(w, r, "/custom", http.StatusFound)
	} else {
		AddFlashQuick(w, r, "Custom race started!")
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func customRaceLoadHandler(w http.ResponseWriter, r *http.Request) {
	err := raceManager.StartCustomRace(chi.URLParam(r, "uuid"))

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
