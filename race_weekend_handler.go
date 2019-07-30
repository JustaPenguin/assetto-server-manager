package servermanager

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

type RaceWeekendHandler struct {
	*BaseHandler

	raceWeekendManager *RaceWeekendManager
}

func NewRaceWeekendHandler(baseHandler *BaseHandler, raceWeekendManager *RaceWeekendManager) *RaceWeekendHandler {
	return &RaceWeekendHandler{
		BaseHandler:        baseHandler,
		raceWeekendManager: raceWeekendManager,
	}
}

func (rwh *RaceWeekendHandler) list(w http.ResponseWriter, r *http.Request) {
	raceWeekends, err := rwh.raceWeekendManager.ListRaceWeekends()

	if err != nil {
		logrus.Errorf("couldn't list weekends, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "race-weekend/index.html", map[string]interface{}{
		"RaceWeekends": raceWeekends,
	})
}

func (rwh *RaceWeekendHandler) createOrEdit(w http.ResponseWriter, r *http.Request) {
	raceWeekendOpts, err := rwh.raceWeekendManager.BuildRaceWeekendTemplateOpts(r)

	if err != nil {
		panic(err)
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "race-weekend/new.html", raceWeekendOpts)
}

// submit creates a given Championship and redirects the user to begin
// the flow of adding events to the new Championship
func (rwh *RaceWeekendHandler) submit(w http.ResponseWriter, r *http.Request) {
	raceWeekend, edited, err := rwh.raceWeekendManager.SaveRaceWeekend(r)

	if err != nil {
		logrus.Errorf("couldn't create race weekend, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if edited {
		AddFlash(w, r, "Race Weekend successfully edited!")
		http.Redirect(w, r, "/race-weekend/"+raceWeekend.ID.String(), http.StatusFound)
	} else {
		AddFlash(w, r, "We've created the Race Weekend. Now you need to add some sessions!")
		http.Redirect(w, r, "/race-weekend/"+raceWeekend.ID.String()+"/session", http.StatusFound)
	}
}

func (rwh *RaceWeekendHandler) sessionConfiguration(w http.ResponseWriter, r *http.Request) {
	raceWeekendSessionOpts, err := rwh.raceWeekendManager.BuildRaceWeekendSessionOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build championship race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rwh.viewRenderer.MustLoadTemplate(w, r, "custom-race/new.html", raceWeekendSessionOpts)
}
